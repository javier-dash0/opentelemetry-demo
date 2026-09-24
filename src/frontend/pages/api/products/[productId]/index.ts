// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

import type { NextApiRequest, NextApiResponse } from 'next';
import { status as GrpcStatus } from '@grpc/grpc-js';
import InstrumentationMiddleware from '../../../../utils/telemetry/InstrumentationMiddleware';
import { Empty, Product } from '../../../../protos/demo';
import ProductCatalogService from '../../../../services/ProductCatalog.service';

type TResponse = Product | Empty;

const isNotFound = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && (error as { code?: unknown }).code === GrpcStatus.NOT_FOUND;

const handler = async ({ method, query }: NextApiRequest, res: NextApiResponse<TResponse>) => {
  switch (method) {
    case 'GET': {
      const { productId = '', currencyCode = '' } = query;

      try {
        const product = await ProductCatalogService.getProduct(productId as string, currencyCode as string);

        return res.status(200).json(product);
      } catch (error) {
        // An unknown product ID is a client error. Answer it with 404 instead of
        // letting the gRPC NOT_FOUND bubble up as an unhandled 500, which marks
        // the frontend span as an error and inflates the service error rate.
        if (isNotFound(error)) {
          return res.status(404).send('');
        }

        throw error;
      }
    }

    default: {
      return res.status(405).send('');
    }
  }
};

export default InstrumentationMiddleware(handler);
