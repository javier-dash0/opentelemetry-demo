// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

import type { NextApiRequest, NextApiResponse } from 'next';
import InstrumentationMiddleware from '../../../../utils/telemetry/InstrumentationMiddleware';
import { Empty, Product } from '../../../../protos/demo';
import ProductCatalogService from '../../../../services/ProductCatalog.service';
import { isGrpcNotFoundError } from '../../../../utils/GrpcErrors';

type TResponse = Product | Empty | { message: string };

const handler = async ({ method, query }: NextApiRequest, res: NextApiResponse<TResponse>) => {
  switch (method) {
    case 'GET': {
      const { productId = '', currencyCode = '' } = query;

      try {
        const product = await ProductCatalogService.getProduct(productId as string, currencyCode as string);

        return res.status(200).json(product);
      } catch (error) {
        // A product that does not exist is a client error, not a server failure.
        if (isGrpcNotFoundError(error)) {
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
