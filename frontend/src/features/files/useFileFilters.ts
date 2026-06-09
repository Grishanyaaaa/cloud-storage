import { useMemo, useState } from "react";
import type { NodeResponse, NodeKind } from "@/api/types";

export type SortField = "name" | "size" | "updated_at";
export type SortDir = "asc" | "desc";
export type KindFilter = "all" | NodeKind;

export interface FileFilters {
  search: string;
  setSearch: (v: string) => void;
  kindFilter: KindFilter;
  setKindFilter: (v: KindFilter) => void;
  sortField: SortField;
  sortDir: SortDir;
  setSort: (field: SortField) => void;
}

export function useFileFilters(): FileFilters {
  const [search, setSearch] = useState("");
  const [kindFilter, setKindFilter] = useState<KindFilter>("all");
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const setSort = (field: SortField) => {
    if (field === sortField) {
      setSortDir((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  return { search, setSearch, kindFilter, setKindFilter, sortField, sortDir, setSort };
}

function compareStrings(a: string, b: string): number {
  return a.localeCompare(b, "ru-RU", { sensitivity: "base" });
}

export function useFilteredItems(items: NodeResponse[], filters: FileFilters): NodeResponse[] {
  const { search, kindFilter, sortField, sortDir } = filters;

  return useMemo(() => {
    const needle = search.trim().toLowerCase();

    let result = items;

    if (needle) {
      result = result.filter((n) => n.name.toLowerCase().includes(needle));
    }

    if (kindFilter !== "all") {
      result = result.filter((n) => n.kind === kindFilter);
    }

    result = [...result].sort((a, b) => {
      // Folders always come first regardless of sort
      if (a.kind !== b.kind) return a.kind === "folder" ? -1 : 1;

      let cmp = 0;
      switch (sortField) {
        case "name":
          cmp = compareStrings(a.name, b.name);
          break;
        case "size":
          cmp = (a.size_bytes ?? 0) - (b.size_bytes ?? 0);
          break;
        case "updated_at":
          cmp = new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
          break;
      }
      return sortDir === "asc" ? cmp : -cmp;
    });

    return result;
  }, [items, search, kindFilter, sortField, sortDir]);
}
