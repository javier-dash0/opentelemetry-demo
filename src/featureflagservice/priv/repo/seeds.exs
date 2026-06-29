# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# This file contains seed data for the Feature Flag Service database.
# It runs after migrations via `mix ecto.setup` or `mix run priv/repo/seeds.exs`.
#
# Feature flags use a float `enabled` field as a probability value:
#   0.0  = always disabled (0% chance of triggering)
#   1.0  = always enabled (100% chance of triggering)
#
# Fault-injection flags are seeded to 0.0 (disabled) by default.
# Enable them intentionally via the Feature Flag Service UI to demonstrate
# failure scenarios. Leaving them enabled at high probability causes
# sustained errors that trigger production alerts.

alias Featureflagservice.FeatureFlags

flags = [
  %{
    "name" => "productCatalogFailure",
    "description" =>
      "Fault injection: causes GetProduct to return an Internal error for product OLJCESPC7Z " <>
        "with a probability equal to the enabled value. Set to 0.0 to disable. " <>
        "When this flag is left enabled at high probability it triggers the " <>
        "'Product Catalog Service High Error Percentage' alert.",
    "enabled" => 0.0
  },
  %{
    "name" => "adServiceFailure",
    "description" =>
      "Fault injection: causes GetAds to return a RESOURCE_EXHAUSTED error with a probability " <>
        "equal to the enabled value. Set to 0.0 to disable.",
    "enabled" => 0.0
  },
  %{
    "name" => "cartServiceFailure",
    "description" =>
      "Fault injection: causes AddItem to fail with an Internal error with a probability " <>
        "equal to the enabled value. Set to 0.0 to disable.",
    "enabled" => 0.0
  },
  %{
    "name" => "paymentServiceFailure",
    "description" =>
      "Fault injection: causes Charge to return an Invalid Argument error with a probability " <>
        "equal to the enabled value. Set to 0.0 to disable.",
    "enabled" => 0.0
  },
  %{
    "name" => "recommendationServiceCacheFailure",
    "description" =>
      "Fault injection: causes the recommendation service cache to malfunction with a probability " <>
        "equal to the enabled value. Set to 0.0 to disable.",
    "enabled" => 0.0
  }
]

Enum.each(flags, fn attrs ->
  case FeatureFlags.get_feature_flag_by_name(attrs["name"]) do
    nil ->
      {:ok, _} = FeatureFlags.create_feature_flag(attrs)
      IO.puts("Seeded feature flag: #{attrs["name"]} (enabled=#{attrs["enabled"]})")

    existing ->
      IO.puts("Skipping existing feature flag: #{existing.name} (enabled=#{existing.enabled})")
  end
end)
