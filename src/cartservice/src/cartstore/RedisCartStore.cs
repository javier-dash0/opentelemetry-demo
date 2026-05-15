// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
using System;
using System.Linq;
using System.Threading.Tasks;
using Grpc.Core;
using StackExchange.Redis;
using Google.Protobuf;
using Microsoft.Extensions.Logging;

namespace cartservice.cartstore;

public class RedisCartStore : ICartStore
{
    private readonly ILogger _logger;
    private const string CartFieldName = "cart";
    private const int RedisRetryNumber = 30;
    private const int MaxConnectionAttempts = 3;
    private const int InitialRetryDelayMs = 100;

    private volatile ConnectionMultiplexer _redis;
    private volatile bool _isRedisConnectionOpened;

    private readonly object _locker = new();
    private readonly byte[] _emptyCartBytes;
    private readonly string _connectionString;

    private readonly ConfigurationOptions _redisConnectionOptions;

    public RedisCartStore(ILogger<RedisCartStore> logger, string redisAddress)
    {
        _logger = logger;
        // Serialize empty cart into byte array.
        var cart = new Oteldemo.Cart();
        _emptyCartBytes = cart.ToByteArray();
        _connectionString = $"{redisAddress},ssl=false,allowAdmin=true,abortConnect=false";

        _redisConnectionOptions = ConfigurationOptions.Parse(_connectionString);

        // Try to reconnect multiple times if the first retry fails.
        _redisConnectionOptions.ConnectRetry = RedisRetryNumber;
        _redisConnectionOptions.ReconnectRetryPolicy = new ExponentialRetry(1000);

        _redisConnectionOptions.KeepAlive = 180;
        _redisConnectionOptions.ConnectTimeout = 5000; // 5 second connection timeout
        _redisConnectionOptions.SyncTimeout = 5000; // 5 second operation timeout
        _redisConnectionOptions.AsyncTimeout = 5000;
    }

    public ConnectionMultiplexer GetConnection()
    {
        EnsureRedisConnected();
        return _redis;
    }

    public void Initialize()
    {
        EnsureRedisConnected();
    }

    private void EnsureRedisConnected()
    {
        if (_isRedisConnectionOpened && _redis != null && _redis.IsConnected)
        {
            return;
        }

        // Connection is closed or failed - open a new one but only at the first thread
        lock (_locker)
        {
            if (_isRedisConnectionOpened && _redis != null && _redis.IsConnected)
            {
                return;
            }

            // Implement retry logic with exponential backoff
            Exception lastException = null;
            for (int attempt = 1; attempt <= MaxConnectionAttempts; attempt++)
            {
                try
                {
                    _logger.LogDebug("Connecting to Redis (attempt {attempt}/{MaxConnectionAttempts}): {_connectionString}", 
                        attempt, MaxConnectionAttempts, _connectionString);
                    
                    _redis = ConnectionMultiplexer.Connect(_redisConnectionOptions);

                    if (_redis == null || !_redis.IsConnected)
                    {
                        throw new ApplicationException("Connection multiplexer returned null or not connected");
                    }

                    _logger.LogInformation("Successfully connected to Redis on attempt {attempt}", attempt);
                    var cache = _redis.GetDatabase();

                    _logger.LogDebug("Performing connection validation test");
                    cache.StringSet("cart", "OK");
                    object res = cache.StringGet("cart");
                    _logger.LogDebug("Connection validation test result: {res}", res);

                    _redis.InternalError += (_, e) => { 
                        _logger.LogError(e.Exception, "Redis internal error occurred");
                    };
                    _redis.ConnectionRestored += (_, _) =>
                    {
                        _isRedisConnectionOpened = true;
                        _logger.LogInformation("Connection to redis was restored successfully.");
                    };
                    _redis.ConnectionFailed += (_, e) =>
                    {
                        _logger.LogWarning("Connection failed: {FailureType}. Will retry on next operation.", e.FailureType);
                        _isRedisConnectionOpened = false;
                    };

                    _isRedisConnectionOpened = true;
                    return; // Successfully connected
                }
                catch (Exception ex)
                {
                    lastException = ex;
                    _logger.LogWarning(ex, "Failed to connect to Redis on attempt {attempt}/{MaxConnectionAttempts}", 
                        attempt, MaxConnectionAttempts);
                    
                    _isRedisConnectionOpened = false;
                    
                    // Dispose failed connection attempt
                    if (_redis != null)
                    {
                        try
                        {
                            _redis.Dispose();
                        }
                        catch (Exception disposeEx)
                        {
                            _logger.LogDebug(disposeEx, "Error disposing failed connection");
                        }
                        _redis = null;
                    }
                    
                    // Don't wait after the last attempt
                    if (attempt < MaxConnectionAttempts)
                    {
                        int delayMs = InitialRetryDelayMs * (int)Math.Pow(2, attempt - 1);
                        _logger.LogDebug("Waiting {delayMs}ms before retry", delayMs);
                        Task.Delay(delayMs).Wait();
                    }
                }
            }

            // All connection attempts failed
            _logger.LogError(lastException, "Failed to connect to Redis after {MaxConnectionAttempts} attempts", MaxConnectionAttempts);
            throw new ApplicationException($"Wasn't able to connect to redis after {MaxConnectionAttempts} attempts", lastException);
        }
    }

    public async Task AddItemAsync(string userId, string productId, int quantity)
    {
        _logger.LogInformation("AddItemAsync called with userId={userId}, productId={productId}, quantity={quantity}", userId, productId, quantity);

        try
        {
            EnsureRedisConnected();

            var db = _redis.GetDatabase();

            // Access the cart from the cache
            var value = await db.HashGetAsync(userId, CartFieldName);

            Oteldemo.Cart cart;
            if (value.IsNull)
            {
                cart = new Oteldemo.Cart
                {
                    UserId = userId
                };
                cart.Items.Add(new Oteldemo.CartItem { ProductId = productId, Quantity = quantity });
            }
            else
            {
                cart = Oteldemo.Cart.Parser.ParseFrom(value);
                var existingItem = cart.Items.SingleOrDefault(i => i.ProductId == productId);
                if (existingItem == null)
                {
                    cart.Items.Add(new Oteldemo.CartItem { ProductId = productId, Quantity = quantity });
                }
                else
                {
                    existingItem.Quantity += quantity;
                }
            }

            await db.HashSetAsync(userId, new[]{ new HashEntry(CartFieldName, cart.ToByteArray()) });
            await db.KeyExpireAsync(userId, TimeSpan.FromMinutes(60));
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, $"Can't access cart storage. {ex}"));
        }
    }

    public async Task EmptyCartAsync(string userId)
    {
        _logger.LogInformation("EmptyCartAsync called with userId={userId}", userId);

        try
        {
            EnsureRedisConnected();
            var db = _redis.GetDatabase();

            // Update the cache with empty cart for given user
            await db.HashSetAsync(userId, new[] { new HashEntry(CartFieldName, _emptyCartBytes) });
            await db.KeyExpireAsync(userId, TimeSpan.FromMinutes(60));
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, $"Can't access cart storage. {ex}"));
        }
    }

    public async Task<Oteldemo.Cart> GetCartAsync(string userId)
    {
        _logger.LogInformation("GetCartAsync called with userId={userId}", userId);

        try
        {
            EnsureRedisConnected();

            var db = _redis.GetDatabase();

            // Access the cart from the cache
            var value = await db.HashGetAsync(userId, CartFieldName);

            if (!value.IsNull)
            {
                return Oteldemo.Cart.Parser.ParseFrom(value);
            }

            // We decided to return empty cart in cases when user wasn't in the cache before
            return new Oteldemo.Cart();
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, $"Can't access cart storage. {ex}"));
        }
    }

    public bool Ping()
    {
        try
        {
            var cache = _redis.GetDatabase();
            var res = cache.Ping();
            return res != TimeSpan.Zero;
        }
        catch (Exception)
        {
            return false;
        }
    }
}
