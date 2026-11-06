import { useEffect, useState } from "react";
import { feedback, type Reason } from "./session";

export type Resource<T> =
  | { kind: "loading" }
  | { kind: "failed"; reason: Reason }
  | { kind: "ready"; data: T };
export function useResource<T>(load: (signal: AbortSignal) => Promise<T>) {
  const [tick, setTick] = useState(0);
  const [result, setResult] = useState<{
    owner: typeof load;
    tick: number;
    value: Resource<T>;
  }>();
  useEffect(() => {
    const c = new AbortController();
    load(c.signal)
      .then((data) => {
        if (!c.signal.aborted)
          setResult({ owner: load, tick, value: { kind: "ready", data } });
      })
      .catch((error) => {
        if (!c.signal.aborted)
          setResult({
            owner: load,
            tick,
            value: { kind: "failed", reason: feedback(error) },
          });
      });
    return () => c.abort();
  }, [load, tick]);
  const state: Resource<T> =
    result?.owner === load && result.tick === tick
      ? result.value
      : { kind: "loading" };
  return { state, refresh: () => setTick((n) => n + 1) };
}
