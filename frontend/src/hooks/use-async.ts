import { useEffect, useRef, useState, useCallback } from "react";
import type { DependencyList } from "react";

export type AsyncState<T> = {
  data: T | null;
  loading: boolean;
  error: Error | null;
};

/**
 * useAsync - Automatically fetches data on mount or when dependencies change
 *
 * Similar to Flutter's FutureBuilder - handles loading, error, and data states.
 *
 * @param asyncFn - The async function to execute
 * @param dependencies - Array of dependencies that trigger refetch when changed
 * @returns AsyncState with data, loading, and error
 *
 * @example
 * const { data, loading, error } = useAsync(listUsers);
 * const { data: user } = useAsync(() => getUserDetails(uuid), [uuid]);
 */
export function useAsync<T>(
  asyncFn: () => Promise<T>,
  dependencies: DependencyList = [],
): AsyncState<T> {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: true,
    error: null,
  });

  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    setState({ data: null, loading: true, error: null });

    asyncFn()
      .then((data) => {
        if (mountedRef.current) {
          setState({ data, loading: false, error: null });
        }
      })
      .catch((error) => {
        if (mountedRef.current) {
          setState({ data: null, loading: false, error });
        }
      });

    return () => {
      mountedRef.current = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, dependencies);

  return state;
}

/**
 * useAsyncFn - Manual trigger for async operations (mutations, form submissions)
 *
 * Returns a state object and an execute function to trigger the async operation.
 *
 * @param asyncFn - The async function to execute
 * @returns [state, execute, reset] tuple
 *
 * @example
 * const [{ loading, error }, executeDelete, reset] = useAsyncFn(deleteUser);
 * await executeDelete(userUuid);
 */
export function useAsyncFn<T, Args extends unknown[]>(
  asyncFn: (...args: Args) => Promise<T>,
): [AsyncState<T>, (...args: Args) => Promise<void>, () => void] {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: false,
    error: null,
  });

  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const execute = useCallback(
    async (...args: Args) => {
      setState({ data: null, loading: true, error: null });
      try {
        const data = await asyncFn(...args);
        if (mountedRef.current) {
          setState({ data, loading: false, error: null });
        }
      } catch (error) {
        if (mountedRef.current) {
          setState({ data: null, loading: false, error: error as Error });
        }
      }
    },
    [asyncFn],
  );

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null });
  }, []);

  return [state, execute, reset];
}
