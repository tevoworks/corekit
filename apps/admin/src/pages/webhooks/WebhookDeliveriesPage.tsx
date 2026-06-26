import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { Webhook, WebhookDelivery } from '../../lib/types'
import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, PageHeader, Table, Button, StatusBadge, EmptyState, LoadingSkeleton } from '../../components/ui'

export default function WebhookDeliveriesPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [cursor, setCursor] = useState(0)
  const [allDeliveries, setAllDeliveries] = useState<WebhookDelivery[]>([])
  const [expanded, setExpanded] = useState<number | null>(null)
  const [nextCursor, setNextCursor] = useState(0)

  const whId = Number(id)

  const { data: webhook } = useQuery({
    queryKey: ['webhook', whId],
    queryFn: () => api.get(`/api/webhooks/${whId}`).then(r => r.data.data as Webhook),
    enabled: !!whId,
  })

  const { isLoading } = useQuery({
    queryKey: ['deliveries', whId, cursor],
    queryFn: () => api.get(`/api/webhooks/${whId}/deliveries?limit=20&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data as WebhookDelivery[] : []
      if (cursor === 0) {
        setAllDeliveries(items)
      } else {
        setAllDeliveries(prev => [...prev, ...items])
      }
      setNextCursor(r.data.meta?.next_cursor || 0)
      return items
    }),
    enabled: !!whId,
  })

  const retryMutation = useMutation({
    mutationFn: (deliveryId: number) => api.post(`/api/webhooks/${whId}/deliveries/${deliveryId}/retry`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['deliveries', whId] })
    },
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title={webhook ? `Deliveries: ${webhook.name}` : 'Deliveries'}
        description={webhook ? `Webhook: ${webhook.url}` : undefined}
        action={
          <Button variant="ghost" icon="arrow_back" onClick={() => navigate('/webhooks')}>
            Back to Webhooks
          </Button>
        }
      />

      {isLoading && allDeliveries.length === 0 ? (
        <Card testId="deliveries-loading-card"><LoadingSkeleton rows={5} testId="deliveries-loading" /></Card>
      ) : allDeliveries.length === 0 ? (
        <Card testId="deliveries-empty-card">
          <EmptyState icon="inbox" title="No deliveries yet" description="Deliveries will appear here when events are triggered." testId="deliveries-empty-state" />
        </Card>
      ) : (
        <>
          <Table headers={['Event', 'Status', 'Code', 'Duration', 'Time', '']} testId="deliveries-table">
            {allDeliveries.map((d: WebhookDelivery) => (
              <tr key={d.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
                <td className="p-3 font-mono text-xs">{d.event}</td>
                <td className="p-3"><StatusBadge status={d.status} /></td>
                <td className="p-3 text-xs">{d.response_code ?? '\u2014'}</td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)]">
                  {d.duration_ms != null ? `${d.duration_ms}ms` : '\u2014'}
                </td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)]">{new Date(d.created_at).toLocaleString()}</td>
                <td className="p-3 text-right">
                  <div className="flex gap-1 justify-end">
                    {d.status === 'failed' && (
                      <Button variant="secondary" size="sm" icon="refresh"
                        onClick={() => retryMutation.mutate(d.id)}
                        loading={retryMutation.isPending && retryMutation.variables === d.id}
                        data-testid={`delivery-retry-${d.id}`}
                      >
                        Retry
                      </Button>
                    )}
                    <Button variant="ghost" size="sm"
                      onClick={() => setExpanded(expanded === d.id ? null : d.id)}
                      data-testid={`delivery-expand-${d.id}`}
                    >
                      {expanded === d.id ? 'Less' : 'Details'}
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </Table>

          {expanded && (
            <DeliveryDetail
              delivery={allDeliveries.find(d => d.id === expanded)!}
              onClose={() => setExpanded(null)}
            />
          )}

          {nextCursor > 0 && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(nextCursor)} data-testid="deliveries-load-more-button">
                Load More
              </Button>
            </div>
          )}
        </>
      )}
    </main>
  )
}

function DeliveryDetail({ delivery, onClose }: { delivery: WebhookDelivery; onClose: () => void }) {
  return (
    <Card className="mt-4" testId="delivery-detail-card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold">Delivery Details</h3>
        <Button variant="ghost" size="sm" onClick={onClose} data-testid="delivery-detail-close-button">Close</Button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {delivery.error_message && (
          <div className="md:col-span-2">
            <label className="block text-xs font-medium text-[var(--on-surface-variant)] mb-1">Error</label>
            <div className="p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-xs font-mono whitespace-pre-wrap break-all">
              {delivery.error_message}
            </div>
          </div>
        )}
        <div>
          <label className="block text-xs font-medium text-[var(--on-surface-variant)] mb-1">Request Body</label>
          <pre className="p-3 rounded-lg bg-[var(--surface-container)] text-xs font-mono whitespace-pre-wrap break-all max-h-48 overflow-y-auto">
            {delivery.request_body ? JSON.stringify(JSON.parse(delivery.request_body), null, 2) : '\u2014'}
          </pre>
        </div>
        <div>
          <label className="block text-xs font-medium text-[var(--on-surface-variant)] mb-1">Response Body</label>
          <pre className="p-3 rounded-lg bg-[var(--surface-container)] text-xs font-mono whitespace-pre-wrap break-all max-h-48 overflow-y-auto">
            {delivery.response_body ?? '\u2014'}
          </pre>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-4 mt-3">
        <div>
          <label className="block text-xs font-medium text-[var(--on-surface-variant)] mb-1">Status</label>
          <StatusBadge status={delivery.status} />
        </div>
        <div>
          <label className="block text-xs font-medium text-[var(--on-surface-variant)] mb-1">Response Code</label>
          <span className="text-sm">{delivery.response_code ?? '\u2014'}</span>
        </div>
        <div>
          <label className="block text-xs font-medium text-[var(--on-surface-variant)] mb-1">Duration</label>
          <span className="text-sm">{delivery.duration_ms != null ? `${delivery.duration_ms}ms` : '\u2014'}</span>
        </div>
      </div>
    </Card>
  )
}
