/*
  CSV 导出工具
  功能：将表格数据转换为 CSV 格式并触发浏览器下载
*/

interface CsvColumn<T> {
  header: string
  accessor: (row: T) => string | number
}

export function exportCsv<T>(
  data: T[],
  columns: CsvColumn<T>[],
  filename: string
) {
  const BOM = "\uFEFF"
  const headerRow = columns.map((c) => `"${c.header}"`).join(",")
  const dataRows = data.map((row) =>
    columns
      .map((c) => {
        const val = c.accessor(row)
        return typeof val === "string" ? `"${val.replace(/"/g, '""')}"` : val
      })
      .join(",")
  )

  const csv = BOM + [headerRow, ...dataRows].join("\n")
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" })
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = `${filename}_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}
