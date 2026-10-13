import { useEffect, useState } from "react";
import { ApiError } from "../api/client";

/**
 * One read of the UI API, reloaded when `key` changes and aborted when it
 * changes again. A 404 is not an error here: the explore routes (D744) answer
 * it to a principal that may not see that part of the KB, and the static demo
 * answers it for what it did not export. Either way the page leaves the part
 * out (`absent`) rather than showing a failure.
 */
export interface Resource<T> {
  data: T | null;
  error: unknown;
  loading: boolean;
  absent: boolean;
}

export function useResource<T>(
  key: string | null,
  load: (signal: AbortSignal) => Promise<T>,
  onFailure?: (err: unknown) => boolean,
): Resource<T> {
  const [state, setState] = useState<Resource<T>>({ data: null, error: null, loading: key !== null, absent: false });

  useEffect(() => {
    if (key === null) {
      setState({ data: null, error: null, loading: false, absent: true });
      return;
    }
    const controller = new AbortController();
    setState((s) => ({ ...s, loading: true }));
    load(controller.signal)
      .then((data) => {
        if (!controller.signal.aborted) setState({ data, error: null, loading: false, absent: false });
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        if (err instanceof ApiError && err.isNotFound) {
          setState({ data: null, error: null, loading: false, absent: true });
          return;
        }
        if (onFailure?.(err)) return;
        setState({ data: null, error: err, loading: false, absent: false });
      });
    return () => controller.abort();
    // `load` is a fresh closure on every render: the key says when to reload.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  return state;
}
