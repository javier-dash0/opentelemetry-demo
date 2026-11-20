// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
using System.Collections.Generic;
using System.Threading.Tasks;
using cartservice.cartstore;
using cartservice.featureflags;
using Grpc.Core;
using Oteldemo;
using Xunit;

namespace cartservice.tests;

public class CartServiceQuantityTests
{
    [Theory]
    [InlineData(0)]
    [InlineData(-1)]
    public async Task AddItem_QuantityBelowOne_RejectedWithInvalidArgument(int quantity)
    {
        var store = new RecordingCartStore();
        var service = CreateService(store);
        var request = new AddItemRequest
        {
            UserId = "user-1",
            Item = new CartItem { ProductId = "OLJCESPC7Z", Quantity = quantity },
        };

        var exception = await Assert.ThrowsAsync<RpcException>(() => service.AddItem(request, null));

        Assert.Equal(StatusCode.InvalidArgument, exception.StatusCode);
        Assert.Empty(store.AddedItems);
    }

    [Fact]
    public async Task AddItem_QuantityOne_Stored()
    {
        var store = new RecordingCartStore();
        var service = CreateService(store);
        var request = new AddItemRequest
        {
            UserId = "user-1",
            Item = new CartItem { ProductId = "OLJCESPC7Z", Quantity = 1 },
        };

        await service.AddItem(request, null);

        var added = Assert.Single(store.AddedItems);
        Assert.Equal(("user-1", "OLJCESPC7Z", 1), added);
    }

    private static cartservice.services.CartService CreateService(RecordingCartStore store) =>
        new(store, store, new FeatureFlagHelper());

    private sealed class RecordingCartStore : ICartStore
    {
        public List<(string UserId, string ProductId, int Quantity)> AddedItems { get; } = new();

        public void Initialize()
        {
        }

        public Task AddItemAsync(string userId, string productId, int quantity)
        {
            AddedItems.Add((userId, productId, quantity));
            return Task.CompletedTask;
        }

        public Task EmptyCartAsync(string userId) => Task.CompletedTask;

        public Task<Cart> GetCartAsync(string userId) => Task.FromResult(new Cart());

        public bool Ping() => true;
    }
}
