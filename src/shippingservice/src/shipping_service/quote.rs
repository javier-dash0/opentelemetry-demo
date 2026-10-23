// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

use core::fmt;
use std::{collections::HashMap, env};

use log::debug;
use opentelemetry::global;
use opentelemetry::{trace::get_active_span, Context, KeyValue};
use opentelemetry_http::HeaderInjector;
use reqwest::header::HeaderMap;
use reqwest_middleware::ClientBuilder;
use reqwest_tracing::{TracingMiddleware, SpanBackendWithUrl};

use reqwest::Method;

#[derive(Debug, Default)]
pub struct Quote {
    pub dollars: i64,
    pub cents: i32,
}

// TODO: Check product catalog for price on each item (will likley need item ID)
pub async fn create_quote_from_count(count: u32) -> Result<Quote, tonic::Status> {
    let f = match request_quote(count).await {
        Ok(float) => float,
        Err(err) => {
            let msg = format!("{}", err);
            return Err(tonic::Status::unknown(msg));
        }
    };

    Ok(get_active_span(|span| {
        let q = create_quote_from_float(f);
        span.add_event(
            "Received Quote".to_string(),
            vec![KeyValue::new("app.shipping.cost.total", format!("{}", q))],
        );
        span.set_attribute(KeyValue::new("app.shipping.items.count", count as i64));
        span.set_attribute(KeyValue::new("app.shipping.cost.total", format!("{}", q)));
        q
    }))
}

async fn request_quote(count: u32) -> Result<f64, Box<dyn std::error::Error>> {
    // TODO: better testing here and default quote_service_addr
    let quote_service_addr: String = format!(
        "{}{}",
        env::var("QUOTE_SERVICE_ADDR").expect("$QUOTE_SERVICE_ADDR is not set"),
        "/getquote"
    );

    let mut reqbody = HashMap::new();
    reqbody.insert("numberOfItems", count);

    let client = ClientBuilder::new(reqwest::Client::new())
        .with(TracingMiddleware::<SpanBackendWithUrl>::new())
        .build();

    let req = client.request(Method::POST, quote_service_addr);

    let mut headers = HeaderMap::new();

    let cx = Context::current();
    global::get_text_map_propagator(|propagator| {
        propagator.inject_context(&cx, &mut HeaderInjector(&mut headers))
    });

    let resp = req
        .json(&reqbody)
        .headers(headers)
        .send()
        .await?
        .text_with_charset("utf-8")
        .await?;

    debug!("{:?}", resp);

    match resp.parse::<f64>() {
        Ok(f) => Ok(f),
        Err(error) => Err(Box::new(error)),
    }
}

pub fn create_quote_from_float(value: f64) -> Quote {
    Quote {
        dollars: value.floor() as i64,
        cents: ((value * 100_f64).round() as i32) % 100,
    }
}

impl fmt::Display for Quote {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{}.{}", self.dollars, self.cents)
    }
}

#[cfg(test)]
mod tests {
    use super::create_quote_from_float;

    #[test]
    fn quote_keeps_the_exact_cent_from_the_service() {
        // The quote service returns round(8.90 * count, 2) as a decimal string.
        // For count = 9 it returns 80.10, which parses to the f64 80.099999999999994.
        // The conversion must keep the cent the service charged.
        let q = create_quote_from_float(80.10_f64);
        assert_eq!(q.dollars, 80);
        assert_eq!(q.cents, 10);
    }

    #[test]
    fn quote_keeps_every_cent_for_counts_that_used_to_lose_one() {
        for count in [9_u32, 18, 31, 36, 62, 67, 72, 119, 124, 129, 134, 139, 144] {
            let value = (8.90_f64 * count as f64 * 100.0).round() / 100.0;
            let q = create_quote_from_float(value);
            let expected_total_cents = (value * 100.0).round() as i64;
            assert_eq!(
                q.dollars * 100 + q.cents as i64,
                expected_total_cents,
                "count {} produced {}.{:02}",
                count,
                q.dollars,
                q.cents
            );
        }
    }
}
