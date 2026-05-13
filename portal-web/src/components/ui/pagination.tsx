interface PaginationProps {
  page: number
  limit: number
  total: number
  onPageChange: (page: number) => void
}

export function Pagination({ page, limit, total, onPageChange }: PaginationProps) {
  const totalPages = limit > 0 ? Math.ceil(total / limit) : 0

  const start = Math.max(1, page - 2)
  const end = Math.min(totalPages, start + 4)
  const pageNumbers: number[] = []
  for (let i = start; i <= end; i++) {
    pageNumbers.push(i)
  }

  return (
    <div className="flex items-center gap-1 justify-end pt-2 text-sm text-gray-600">
      {totalPages > 1 && (
        <>
          <button
            aria-label="이전 페이지"
            className="px-2 py-1 rounded border hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed"
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
          >
            &lt;
          </button>

          {pageNumbers.map((p) => (
            <button
              key={p}
              aria-label={`${p}페이지`}
              aria-current={p === page ? 'page' : undefined}
              className={`px-3 py-1 rounded border ${
                p === page
                  ? 'bg-blue-500 text-white border-blue-500'
                  : 'hover:bg-gray-100'
              }`}
              onClick={() => onPageChange(p)}
            >
              {p}
            </button>
          ))}

          <button
            aria-label="다음 페이지"
            className="px-2 py-1 rounded border hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed"
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
          >
            &gt;
          </button>
        </>
      )}
      <span className="ml-2 text-gray-500">총 {total}건</span>
    </div>
  )
}
