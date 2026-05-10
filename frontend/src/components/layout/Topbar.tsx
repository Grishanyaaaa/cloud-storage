import { Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAIModalStore } from "@/store/ai.store";
import { UserMenu } from "./UserMenu";

export function Topbar() {
  const openAi = useAIModalStore((s) => s.open);

  return (
    <div className="flex h-full items-center justify-between gap-4 px-4">
      <div className="flex items-center gap-2 min-w-0 flex-1">
        {/*
         * Search input is intentionally omitted — storage-service has no
         * search endpoint yet (per blueprint Appendix C). We keep this slot
         * for future server-side search.
         */}
      </div>
      <div className="flex items-center gap-2">
        <Button intent="ghost" size="sm" onClick={openAi}>
          <Sparkles className="h-4 w-4" />
          ИИ
        </Button>
        <UserMenu />
      </div>
    </div>
  );
}
