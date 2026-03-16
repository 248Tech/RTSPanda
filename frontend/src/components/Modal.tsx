import React from 'react'

type ModalSize = 'sm' | 'md' | 'lg' | 'xl'

export interface ModalProps {
  title?: string
  children: React.ReactNode
  onClose: () => void
  size?: ModalSize
}

const SIZE_CLASSES: Record<ModalSize, string> = {
  sm: 'max-w-md',
  md: 'max-w-xl',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
}

export function Modal({ title, children, onClose, size = 'sm' }: ModalProps) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? 'modal-title' : undefined}
      className="fixed inset-0 z-50 overflow-y-auto bg-black/70 p-3 backdrop-blur-sm sm:p-4"
      onClick={onClose}
    >
      <div className="flex min-h-full items-start justify-center py-1 sm:items-center sm:py-0">
        <div
          className={`flex w-full ${SIZE_CLASSES[size]} max-h-[calc(100dvh-1rem)] flex-col overflow-hidden rounded-xl border border-border-muted bg-surface shadow-modal sm:max-h-[calc(100dvh-2rem)]`}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between border-b border-border-muted px-5 py-3.5">
            {title ? (
              <h2 id="modal-title" className="text-base font-semibold text-text-primary">
                {title}
              </h2>
            ) : (
              <span />
            )}
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
              aria-label="Close"
            >
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div className="min-h-0 overflow-y-auto px-5 py-4">{children}</div>
        </div>
      </div>
    </div>
  )
}
