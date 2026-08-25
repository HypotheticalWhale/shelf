"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { Search } from "lucide-react";

export function SearchField() {
  const router = useRouter();
  const params = useSearchParams();
  const [value, setValue] = useState(params.get("q") ?? "");

  // Keep the field in step when navigation changes the query behind it.
  useEffect(() => {
    setValue(params.get("q") ?? "");
  }, [params]);

  return (
    <form
      role="search"
      onSubmit={(event) => {
        event.preventDefault();
        const q = value.trim();
        router.push(q ? `/games?q=${encodeURIComponent(q)}` : "/games");
      }}
      className="relative max-w-xs ml-auto sm:ml-0"
    >
      <Search
        className="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-chalk-faint"
        aria-hidden
      />
      <input
        type="search"
        value={value}
        onChange={(event) => setValue(event.target.value)}
        placeholder="Search games"
        aria-label="Search games"
        className="w-full bg-felt-800 border border-rule-soft rounded-full pl-9 pr-3 py-1.5 text-sm placeholder:text-chalk-faint focus:border-rule outline-none transition-colors"
      />
    </form>
  );
}
