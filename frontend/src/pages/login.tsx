import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Field, FieldLabel, FieldError } from "@/components/ui/field";
import { Spinner } from "@/components/ui/spinner";
import { useAuth, ApiError } from "@/contexts/auth-context";

type LoginStep = "credentials" | "totp";

export function LoginPage() {
  const { login } = useAuth();
  const [step, setStep] = useState<LoginStep>("credentials");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [totp, setTotp] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const handleCredentialsSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setPending(true);

    try {
      await login({ username, password });
    } catch (err) {
      if (err instanceof ApiError && err.requiresTotp) {
        setStep("totp");
        setError(null);
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setPending(false);
    }
  };

  const handleTotpSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setPending(true);

    try {
      await login({ username, password, totp });
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setPending(false);
    }
  };

  const handleBackToCredentials = () => {
    setStep("credentials");
    setTotp("");
    setError(null);
  };

  if (step === "totp") {
    return (
      <div className="flex flex-col justify-center items-center min-h-screen">
        <Card className="w-[320px]">
          <CardHeader>
            <CardTitle>Two-Factor Authentication</CardTitle>
          </CardHeader>
          <form onSubmit={handleTotpSubmit}>
            <CardContent className="flex flex-col gap-4">
              <p className="text-sm text-muted-foreground">
                Enter the 6-digit code from your authenticator app.
              </p>
              <Field>
                <FieldLabel htmlFor="totp">Code</FieldLabel>
                <Input
                  id="totp"
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  placeholder="000000"
                  value={totp}
                  onChange={(e) => setTotp(e.target.value)}
                  maxLength={6}
                  disabled={pending}
                  autoFocus
                />
                {error && <FieldError>{error}</FieldError>}
              </Field>
            </CardContent>
            <CardFooter className="flex-col gap-2">
              <Button
                type="submit"
                className="w-full"
                disabled={pending || totp.length !== 6}
              >
                {pending ? <Spinner /> : "Verify"}
              </Button>
              <Button
                type="button"
                variant="ghost"
                className="w-full"
                onClick={handleBackToCredentials}
                disabled={pending}
              >
                Back
              </Button>
            </CardFooter>
          </form>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col justify-center items-center min-h-screen">
      <Card className="w-[320px]">
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
        </CardHeader>
        <form onSubmit={handleCredentialsSubmit}>
          <CardContent className="flex flex-col gap-4">
            <Field>
              <FieldLabel htmlFor="username">Username</FieldLabel>
              <Input
                id="username"
                type="text"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                disabled={pending}
                autoFocus
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="password">Password</FieldLabel>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={pending}
              />
              {error && <FieldError>{error}</FieldError>}
            </Field>
          </CardContent>
          <CardFooter>
            <Button
              type="submit"
              className="w-full"
              disabled={pending || !username || !password}
            >
              {pending ? <Spinner /> : "Sign in"}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
