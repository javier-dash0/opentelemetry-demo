// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

import { ServiceError, status } from '@grpc/grpc-js';
import type { NextApiRequest, NextApiResponse } from 'next';
import InstrumentationMiddleware from '../../../../utils/telemetry/InstrumentationMiddleware';
import { Empty, Product } from '../../../../protos/demo';
import ProductCatalogService from '../../../../services/ProductCatalog.service';

type TResponse = Product | Empty | { message: string };

const handler = async ({ method, query }: NextApiRequest, res: NextApiResponse<TResponse>) => {
  switch (method) {
    case 'GET': {
      const { productId = '', currencyCode = '' } = query;

      try {
        const product = await ProductCatalogService.getProduct(productId as string, currencyCode as string);

        return res.status(200).json(product);
      } catch (error) {
        // ProductCatalogService answers with gRPC NOT_FOUND for an unknown product id.
        // That is a client error, so it has to surface as HTTP 404. Letting it bubble up
        // reaches InstrumentationMiddleware, which turns every thrown error into a 500
        // and marks the server span as ERROR, so a request for a product that simply
        // does not exist is reported as a frontend server fault.
        if ((error as ServiceError)?.code === status.NOT_FOUND) {
          return res.status(404).json({ message: `Product not found: ${productId}` });
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
