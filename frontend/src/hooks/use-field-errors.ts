import { ApiError } from "@/lib/api/client";

/**
 * useFieldErrors - Extracts field-level errors from ApiError
 *
 * Converts server error responses with field keys (e.g., ".username", ".password")
 * into a simple Record<string, string> for easy form validation.
 *
 * @param error - The error from useAsync or useAsyncFn
 * @returns Object mapping field names to error messages
 *
 * @example
 * const [{ error }, execute] = useAsyncFn(createUser);
 * const fieldErrors = useFieldErrors(error);
 *
 * <Input error={fieldErrors.username} />
 * <Input error={fieldErrors.password} />
 */
export function useFieldErrors(error: Error | null): Record<string, string> {
  if (!(error instanceof ApiError)) {
    return {};
  }

  const fieldErrors: Record<string, string> = {};

  error.errorResponse?.errors.forEach((err) => {
    if (err.key) {
      // Remove leading dot from field key (e.g., ".username" -> "username")
      const field = err.key.replace(/^\./, "");

      // Use first error message for each field
      if (!fieldErrors[field]) {
        fieldErrors[field] = err.message;
      }
    }
  });

  return fieldErrors;
}
