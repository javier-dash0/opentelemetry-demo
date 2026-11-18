// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

use std::time::Duration;

use rand::Rng;
use tokio::time::{sleep, timeout};

use opentelemetry::{global, propagation::Extractor, trace::Span, Context, KeyValue};
use opentelemetry::trace::{FutureExt, TraceContextExt, SpanKind, Tracer};
use opentelemetry_semantic_conventions as semcov;
use shop::shipping_service_server::ShippingService;
use tonic::{Request, Response, Status};

use log::*;
use tonic::transport::Channel;

use shop::{GetQuoteRequest, GetQuoteResponse, Money, ShipOrderRequest, ShipOrderResponse};
use crate::shop::feature_flag_service_client::FeatureFlagServiceClient;
use crate::shop::{RangeFeatureFlagRequest, RangeFeatureFlagResponse};

mod quote;
use quote::create_quote_from_count;

mod tracking;
use tracking::create_tracking_id;

const NANOS_MULTIPLE: i32 = 10000000i32;

const RPC_SYSTEM_GRPC: &'static str = "grpc";
const RPC_GRPC_STATUS_CODE_OK: i64 = 0;
const RPC_GRPC_STATUS_CODE_UNKNOWN: i64 = 2;

// The feature flag service is an optional dependency, so a slow or failing
// lookup must not hold up the quote and ship calls: the lookup is bounded.
const SIMULATE_SLOWNESS_FLAG_TIMEOUT: Duration = Duration::from_millis(500);

pub mod shop {
    // The string specified here must match the proto package name
    tonic::include_proto!("oteldemo");
}

#[derive(Debug, Default)]
pub struct ShippingServer {
    feature_flag_client: Option<FeatureFlagServiceClient<Channel>>,
}

struct MetadataMap<'a>(&'a tonic::metadata::MetadataMap);

impl<'a> Extractor for MetadataMap<'a> {
    /// Get a value for a key from the MetadataMap.  If the value can't be converted to &str, returns None
    fn get(&self, key: &str) -> Option<&str> {
        self.0.get(key).and_then(|metadata| metadata.to_str().ok())
    }

    /// Collect all the keys from the MetadataMap.
    fn keys(&self) -> Vec<&str> {
        self.0
            .keys()
            .map(|key| match key {
                tonic::metadata::KeyRef::Ascii(v) => v.as_str(),
                tonic::metadata::KeyRef::Binary(v) => v.as_str(),
            })
            .collect::<Vec<_>>()
    }
}

#[tonic::async_trait]
impl ShippingService for ShippingServer {
    async fn get_quote(
        &self,
        request: Request<GetQuoteRequest>,
    ) -> Result<Response<GetQuoteResponse>, Status> {
        info!("GetQuoteRequest: {:?}", request);
        let parent_cx =
            global::get_text_map_propagator(|prop| prop.extract(&MetadataMap(request.metadata())));

        let request_message = request.into_inner();

        let itemct: u32 = request_message
            .items
            .into_iter()
            .fold(0, |accum, cart_item| accum + (cart_item.quantity as u32));

        // We may want to ask another service for product pricing / info
        // (although now everything is assumed to be the same price)
        // check out the create_quote_from_count method to see how we use the span created here
        let tracer = global::tracer("shippingservice");
        let mut span = tracer.span_builder("oteldemo.ShippingService/GetQuote").with_kind(SpanKind::Server).start_with_context(&tracer, &parent_cx);
        span.set_attribute(semcov::trace::RPC_SYSTEM.string(RPC_SYSTEM_GRPC));

        span.add_event("Processing get quote request".to_string(), vec![]);
        span.set_attribute(KeyValue::new("app.shipping.zip_code", request_message.address.unwrap().zip_code));

        let cx = Context::current_with_span(span);
        let q = match create_quote_from_count(itemct)
            .with_context(cx.clone())
            .await
        {
            Ok(quote) => quote,
            Err(status) => {cx.span().set_attribute(semcov::trace::RPC_GRPC_STATUS_CODE.i64(RPC_GRPC_STATUS_CODE_UNKNOWN)); return Err(status)},
        };

        self.simulate_slowness().await;

        let reply = GetQuoteResponse {
            cost_usd: Some(Money {
                currency_code: "USD".into(),
                units: q.dollars,
                nanos: q.cents * NANOS_MULTIPLE,
            }),
        };
        info!("Sending Quote: {}", q);

        cx.span().set_attribute(semcov::trace::RPC_GRPC_STATUS_CODE.i64(RPC_GRPC_STATUS_CODE_OK));
        Ok(Response::new(reply))
    }
    async fn ship_order(
        &self,
        request: Request<ShipOrderRequest>,
    ) -> Result<Response<ShipOrderResponse>, Status> {
        info!("ShipOrderRequest: {:?}", request);

        let parent_cx =
            global::get_text_map_propagator(|prop| prop.extract(&MetadataMap(request.metadata())));
        // in this case, generating a tracking ID is trivial
        // we'll create a span and associated events all in this function.
        let tracer = global::tracer("shippingservice");
        let mut span = tracer
            .span_builder("oteldemo.ShippingService/ShipOrder").with_kind(SpanKind::Server).start_with_context(&tracer, &parent_cx);
        span.set_attribute(semcov::trace::RPC_SYSTEM.string(RPC_SYSTEM_GRPC));

        span.add_event("Processing shipping order request".to_string(), vec![]);

        self.simulate_slowness().await;

        let tid = create_tracking_id();
        span.set_attribute(KeyValue::new("app.shipping.tracking.id", tid.clone()));
        info!("Tracking ID Created: {}", tid);

        span.add_event(
            "Shipping tracking id created, response sent back".to_string(),
            vec![],
        );

        span.set_attribute(semcov::trace::RPC_GRPC_STATUS_CODE.i64(RPC_GRPC_STATUS_CODE_OK));
        Ok(Response::new(ShipOrderResponse { tracking_id: tid }))
    }
}

impl ShippingServer {

    pub fn new(feature_flag_client: Option<FeatureFlagServiceClient<Channel>>) -> Self {
        Self { feature_flag_client }
    }

    async fn simulate_slowness(&self) {
        if self.feature_flag_client.is_none() {
            return;
        }

        let mut client = self.feature_flag_client.clone().unwrap();

        let request = Request::new(
            RangeFeatureFlagRequest {
                name: String::from("shippingServiceSimulateSlowness"),
                name_lower_bound: String::from("shippingServiceSimulateSlownessLowerBound"),
                name_upper_bound: String::from("shippingServiceSimulateSlownessUpperBound"),
            },
        );

        let response = match timeout(
            SIMULATE_SLOWNESS_FLAG_TIMEOUT,
            client.get_range_feature_flag(request),
        )
        .await
        {
            Ok(Ok(response)) => response,
            Ok(Err(status)) => {
                warn!(
                    "Feature flag service answered the slowness flag with {}, skipping simulated slowness.",
                    status
                );
                return;
            }
            Err(_) => {
                warn!(
                    "Feature flag service did not answer the slowness flag within {:?}, skipping simulated slowness.",
                    SIMULATE_SLOWNESS_FLAG_TIMEOUT
                );
                return;
            }
        };

        sleep_if_slowness_enabled(response.into_inner()).await;
    }
}

// sleep_if_slowness_enabled waits for the delay that the range flag asks for. It
// yields the worker thread while it waits, so other requests served by the same
// worker keep making progress.
async fn sleep_if_slowness_enabled(range_response: RangeFeatureFlagResponse) {
    if !range_response.enabled {
        return;
    }

    // The flag service answers with defaults when a bound is missing, but a
    // configured range can still be empty, which must not panic the request.
    let minimum_delay = range_response.lower_bound.max(0.0).floor();
    let maximum_delay = range_response.upper_bound.max(0.0).floor();
    let delay_millis = if maximum_delay > minimum_delay {
        rand::thread_rng()
            .gen_range(minimum_delay..maximum_delay)
            .floor()
    } else {
        minimum_delay
    };

    let sleep_time = Duration::from_millis(delay_millis as u64);
    info!(
        "Simulating shipping service slowness, waiting {:?}.",
        sleep_time
    );
    sleep(sleep_time).await;
}

#[cfg(test)]
mod tests {
    use super::{
        shop::shipping_service_server::ShippingService,
        shop::{Address, GetQuoteRequest},
        shop::{CartItem, ShipOrderRequest},
        sleep_if_slowness_enabled, ShippingServer, NANOS_MULTIPLE,
    };
    use crate::shop::feature_flag_service_client::FeatureFlagServiceClient;
    use crate::shop::RangeFeatureFlagResponse;
    use std::time::Duration;
    use tonic::transport::Channel;
    use tonic::Request;
    use uuid::Uuid;

    // A channel to a port that nothing listens on: every call on it fails with
    // Unavailable, like a feature flag service that is down.
    fn failing_feature_flag_client() -> FeatureFlagServiceClient<Channel> {
        let channel = Channel::from_static("http://127.0.0.1:1").connect_lazy();
        FeatureFlagServiceClient::new(channel)
    }

    fn make_quote_request_with_items(items: Vec<i32>) -> Request<GetQuoteRequest> {
        let cart_items: Vec<CartItem> = items.into_iter().fold(Vec::new(), |mut accum, count| {
            accum.push(CartItem {
                product_id: "fake-item".to_string(),
                quantity: count,
            });
            accum
        });

        Request::new(GetQuoteRequest {
            address: Some(Address::default()),
            items: cart_items,
        })
    }

    fn make_empty_quote_request() -> Request<GetQuoteRequest> {
        Request::new(GetQuoteRequest::default())
    }

    #[tokio::test]
    async fn empty_quote() {
        let server = ShippingServer::default();

        // when we provide no items, the quote should be empty
        match server.get_quote(make_empty_quote_request()).await {
            Ok(resp) => {
                let money = resp.into_inner().cost_usd.unwrap();
                assert_eq!(money.units, 0);
                assert_eq!(money.nanos, 0);
            }
            Err(e) => panic!("error when making empty quote request: {}", e),
        }
    }

    #[tokio::test]
    async fn quote_for_one_value() {
        let server = ShippingServer::default();

        match server
            .get_quote(make_quote_request_with_items(vec![1_i32]))
            .await
        {
            Ok(resp) => {
                // items are fixed at 8.99, so we should see that price reflected.
                let money = resp.into_inner().cost_usd.unwrap();
                assert_eq!(money.units, 8);
                assert_eq!(money.nanos, 99 * NANOS_MULTIPLE);
            }
            Err(e) => panic!("error when making quote request for one value: {}", e),
        }
    }

    #[tokio::test]
    async fn quote_for_many_values() {
        let server = ShippingServer::default();

        match server
            .get_quote(make_quote_request_with_items(vec![1_i32, 2_i32]))
            .await
        {
            Ok(resp) => {
                // items are fixed at 8.99, so we should see that price reflected for 3 items
                let money = resp.into_inner().cost_usd.unwrap();
                assert_eq!(money.units, 26);
                assert_eq!(money.nanos, 97 * NANOS_MULTIPLE);
            }
            Err(e) => panic!("error when making quote request for many values: {}", e),
        }
    }

    #[tokio::test]
    async fn can_get_tracking_id() {
        let server = ShippingServer::default();

        match server
            .ship_order(Request::new(ShipOrderRequest::default()))
            .await
        {
            Ok(resp) => {
                // we should see a uuid
                match Uuid::parse_str(&resp.into_inner().tracking_id) {
                    Ok(_) => {}
                    Err(e) => panic!("error when parsing uuid: {}", e),
                }
            }
            Err(e) => panic!("error when making request for tracking ID: {}", e),
        }
    }

    #[tokio::test]
    async fn simulate_slowness_survives_feature_flag_service_error() {
        let server = ShippingServer::new(Some(failing_feature_flag_client()));

        // the flag service is optional, so a failing lookup must leave the
        // request alone instead of panicking it.
        server.simulate_slowness().await;
    }

    #[tokio::test]
    async fn ship_order_survives_feature_flag_service_error() {
        let server = ShippingServer::new(Some(failing_feature_flag_client()));

        match server
            .ship_order(Request::new(ShipOrderRequest::default()))
            .await
        {
            Ok(resp) => {
                assert!(Uuid::parse_str(&resp.into_inner().tracking_id).is_ok());
            }
            Err(e) => panic!(
                "error when making ship order request with a failing feature flag service: {}",
                e
            ),
        }
    }

    #[tokio::test(flavor = "current_thread")]
    async fn simulated_slowness_yields_to_other_requests() {
        let flag = RangeFeatureFlagResponse {
            enabled: true,
            lower_bound: 50.0,
            upper_bound: 100.0,
        };
        let mut slowness = Box::pin(sleep_if_slowness_enabled(flag));
        let mut other_request = Box::pin(async {
            tokio::time::sleep(Duration::from_millis(10)).await;
        });

        // the delay must leave the worker thread free to serve other requests.
        tokio::select! {
            _ = &mut other_request => {}
            _ = &mut slowness => panic!(
                "the simulated delay finished before another request made progress"
            ),
        }
    }

    #[tokio::test]
    async fn simulated_slowness_tolerates_an_empty_range() {
        let flag = RangeFeatureFlagResponse {
            enabled: true,
            lower_bound: 1.0,
            upper_bound: 1.0,
        };

        // equal bounds must not panic the request.
        sleep_if_slowness_enabled(flag).await;
    }
}
