// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

import { status } from '@grpc/grpc-js';

/**
 * A gRPC call rejects with a ServiceError carrying a numeric `code`. Only the
 * NOT_FOUND code means "the caller asked for something that does not exist";
 * every other code is a real failure of the downstream service and must keep
 * bubbling up.
 */
export const isGrpcNotFoundError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && (error as { code?: unknown }).code === status.NOT_FOUND;

export default isGrpcNotFoundError;
