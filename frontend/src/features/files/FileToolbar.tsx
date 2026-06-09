import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { FileFilters, KindFilter } from "./useFileFilters";

interface Props {
  filters: FileFilters;
  totalCount: number;
  filteredCount: number;
}

export function FileToolbar({ filters, totalCount, filteredCount }: Props) {
  const hasActiveFilters = filters.search.trim() !== "" || filters.kindFilter !== "all";

  return (
    <div className="flex items-center gap-3 px-4 py-2 border-b border-border-1">
      <div className="relative flex-1 max-w-sm">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-fg-3" />
        <Input
          placeholder="Поиск по имени…"
          value={filters.search}
          onChange={(e) => filters.setSearch(e.target.value)}
          className="pl-8 h-8 text-sm"
        />
        {filters.search && (
          <button
            type="button"
            onClick={() => filters.setSearch("")}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-fg-3 hover:text-fg-1"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      <Select
        value={filters.kindFilter}
        onValueChange={(v) => filters.setKindFilter(v as KindFilter)}
      >
        <SelectTrigger className="w-36 h-8 text-sm">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Все</SelectItem>
          <SelectItem value="folder">Папки</SelectItem>
          <SelectItem value="file">Файлы</SelectItem>
        </SelectContent>
      </Select>

      {hasActiveFilters && totalCount !== filteredCount && (
        <span className="text-xs text-fg-3 whitespace-nowrap">
          {filteredCount} из {totalCount}
        </span>
      )}
    </div>
  );
}
