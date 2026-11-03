import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface Supplier {
  id: number
  name: string
  base_url: string
  status: 'active' | 'inactive'
  notes: string
  account_count: number
  balance: number | null
  balance_status: 'unknown' | 'success' | 'error' | 'unavailable'
  balance_error?: string
  balance_updated_at?: string | null
  balance_checked_at?: string | null
  balance_account_id?: number | null
  has_balance_api_key: boolean
}

export interface SupplierInput {
  name?: string
  base_url?: string
  status?: Supplier['status']
  notes?: string
  balance_account_id?: number | null
  balance_api_key?: string
}

export async function list(page = 1, pageSize = 20, filters: { search?: string; status?: string } = {}): Promise<PaginatedResponse<Supplier>> {
  const { data } = await apiClient.get<PaginatedResponse<Supplier>>('/admin/suppliers', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function getAll(): Promise<Supplier[]> {
  const suppliers: Supplier[] = []
  let page = 1
  while (true) {
    const result = await list(page, 100)
    suppliers.push(...result.items)
    if (suppliers.length >= result.total || result.items.length === 0) return suppliers
    page++
  }
}

export async function getById(id: number): Promise<Supplier> {
  const { data } = await apiClient.get<Supplier>(`/admin/suppliers/${id}`)
  return data
}

export async function create(input: SupplierInput): Promise<Supplier> {
  const { data } = await apiClient.post<Supplier>('/admin/suppliers', input)
  return data
}

export async function update(id: number, input: SupplierInput): Promise<Supplier> {
  const { data } = await apiClient.put<Supplier>(`/admin/suppliers/${id}`, input)
  return data
}

export async function deleteSupplier(id: number): Promise<void> {
  await apiClient.delete(`/admin/suppliers/${id}`)
}

export async function refreshBalance(id: number): Promise<Supplier> {
  const { data } = await apiClient.post<Supplier>(`/admin/suppliers/${id}/refresh-balance`)
  return data
}

export default { list, getAll, getById, create, update, delete: deleteSupplier, refreshBalance }
