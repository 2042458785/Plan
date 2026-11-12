import type { Product } from '@/types/catalog'

export function productImages(product: Product, color: string): string[] {
  const specific = product.colorImages?.[color]
  if (Array.isArray(specific) && specific.length) return specific
  if (product.gallery?.length) return product.gallery
  return product.image ? [product.image] : []
}
