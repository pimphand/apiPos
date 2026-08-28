import http from '@/lib/http';
import type {
  Product,
  CreateProductRequest,
  UpdateProductRequest,
  ListProductsResponse,
  ProductSKU,
} from '@/types';

export const productService = {
  async list(params: {
    page?: number;
    per_page?: number;
    is_active?: string;
    category_id?: string;
    get_sku?: boolean;
  }): Promise<ListProductsResponse> {
    const response = await http.get<ListProductsResponse>('/v1/products', { params });
    return response.data;
  },

  async get(id: string, getSku = false): Promise<Product> {
    const response = await http.get<{ product: Product }>(`/v1/products/${id}`, {
      params: { get_sku: getSku },
    });
    return response.data.product;
  },

  async create(data: CreateProductRequest): Promise<Product> {
    const response = await http.post<{ product: Product }>('/v1/products', data);
    return response.data.product;
  },

  async update(id: string, data: UpdateProductRequest): Promise<Product> {
    const response = await http.put<{ product: Product }>(`/v1/products/${id}`, data);
    return response.data.product;
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/products/${id}`);
  },

  async activate(id: string): Promise<void> {
    await http.post(`/v1/products/${id}/activate`);
  },

  async deactivate(id: string): Promise<void> {
    await http.post(`/v1/products/${id}/deactivate`);
  },

  async search(params: {
    name: string;
    page?: number;
    per_page?: number;
    is_active?: string;
    get_sku?: boolean;
  }): Promise<ListProductsResponse> {
    const response = await http.get<ListProductsResponse>('/v1/products/search', { params });
    return response.data;
  },

  async getLowStock(params: {
    page?: number;
    per_page?: number;
    get_sku?: boolean;
  }): Promise<ListProductsResponse> {
    const response = await http.get<ListProductsResponse>('/v1/products/low-stock', { params });
    return response.data;
  },

  // SKU methods
  async getSKUByBarcode(barcode: string): Promise<ProductSKU> {
    const response = await http.get<{ product_sku: ProductSKU }>(
      `/v1/product-skus/barcode/${barcode}`
    );
    return response.data.product_sku;
  },

  async getSKUBySKU(sku: string): Promise<ProductSKU> {
    const response = await http.get<{ product_sku: ProductSKU }>(
      `/v1/product-skus/sku/${sku}`
    );
    return response.data.product_sku;
  },

  async listSKUs(productId: string, params?: {
    search?: string;
    variant?: string;
    color?: string;
    size?: string;
    page?: number;
    per_page?: number;
  }): Promise<{ product_skus: ProductSKU[]; total_count: number }> {
    const response = await http.get(`/v1/products/${productId}/skus`, { params });
    return response.data;
  },
};
