import { PRIORITY_LABEL, STATUS_LABEL } from '../lib/labels'
import type { Priority, Status } from '../lib/types'

export function StatusBadge({ status }: { status: Status }) {
  return <span className={`badge status-${status}`}>{STATUS_LABEL[status]}</span>
}

export function PriorityBadge({ priority }: { priority: Priority }) {
  return <span className={`badge priority-${priority}`}>{PRIORITY_LABEL[priority]}</span>
}

export function Stars({ value }: { value: number }) {
  return (
    <span className="stars" aria-label={`${value} de 5`}>
      {'★'.repeat(value)}
      <span className="stars-off">{'★'.repeat(5 - value)}</span>
    </span>
  )
}
