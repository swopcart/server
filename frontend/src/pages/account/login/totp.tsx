import { useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ListItem } from "@/components/list-item";
import { Skeleton } from "@/components/ui/skeleton";
import {
  LucideChevronRight,
  LucideShieldCheck,
  LucideShieldOff,
} from "lucide-react";
import { useAuth } from "@/contexts/auth-context";
import {
  getUserDetails,
  generateTOTP,
  enableTOTP,
  disableTOTP,
} from "@/lib/api/users";

export function TotpSettings() {
  const { user } = useAuth();
  const [totpEnabled, setTotpEnabled] = useState<boolean | null>(null);
  const [isLoadingStatus, setIsLoadingStatus] = useState(true);

  useEffect(() => {
    if (!user) return;
    getUserDetails(user.uuid)
      .then((result) => setTotpEnabled(result.totpEnabled))
      .finally(() => setIsLoadingStatus(false));
  }, [user]);

  const handleTotpChange = (enabled: boolean) => {
    setTotpEnabled(enabled);
  };

  if (isLoadingStatus) {
    return <Skeleton className="h-14 w-full rounded-lg" />;
  }

  if (totpEnabled) {
    return <RemoveTotp onDisabled={() => handleTotpChange(false)} />;
  }

  return <SetupTotp onEnabled={() => handleTotpChange(true)} />;
}

interface SetupTotpProps {
  onEnabled: () => void;
}

function SetupTotp({ onEnabled }: SetupTotpProps) {
  const { user } = useAuth();
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState<"password" | "verify">("password");
  const [password, setPassword] = useState("");
  const [secret, setSecret] = useState<string | null>(null);
  const [otpauthUrl, setOtpauthUrl] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const resetState = () => {
    setStep("password");
    setPassword("");
    setSecret(null);
    setOtpauthUrl(null);
    setCode("");
    setError(null);
  };

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
    if (!open) {
      resetState();
    }
  };

  const handlePasswordSubmit = async () => {
    if (!user) return;
    if (!password) {
      setError("Please enter your password");
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const result = await generateTOTP(user.uuid, password);
      setSecret(result.secret);
      setOtpauthUrl(result.url);
      setStep("verify");
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to generate TOTP secret");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleVerify = async () => {
    if (!user || !secret) return;
    if (code.length !== 6) {
      setError("Please enter a 6-digit code");
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      await enableTOTP(user.uuid, secret, code);
      setOpen(false);
      onEnabled();
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to verify code");
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger>
        <ListItem
          title="Set up two-factor authentication"
          leading={<LucideShieldCheck />}
          trailing={<LucideChevronRight />}
        />
      </DialogTrigger>
      <DialogContent>
        <DialogTitle>Set up two-factor authentication</DialogTitle>

        {step === "password" ? (
          <>
            <DialogDescription>
              Enter your password to set up two-factor authentication.
            </DialogDescription>
            <FieldSet>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="setup-password">Password</FieldLabel>
                  <Input
                    id="setup-password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    onKeyDown={(e) =>
                      e.key === "Enter" && handlePasswordSubmit()
                    }
                  />
                </Field>
              </FieldGroup>
              {error && <p className="text-destructive text-sm">{error}</p>}
            </FieldSet>
            <DialogFooter>
              <DialogClose>
                <Button variant="outline">Cancel</Button>
              </DialogClose>
              <Button onClick={handlePasswordSubmit} disabled={isLoading}>
                {isLoading ? "Verifying..." : "Continue"}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogDescription>
              Scan the QR code with your authenticator app, then enter the code
              to verify.
            </DialogDescription>
            <FieldSet>
              <FieldGroup>
                <Field>
                  <FieldLabel>Secret key</FieldLabel>
                  <div className="flex flex-col gap-3">
                    <div className="flex justify-center rounded-md border bg-white p-4">
                      {otpauthUrl && (
                        <QRCodeSVG
                          value={otpauthUrl}
                          size={180}
                          level="M"
                          marginSize={0}
                        />
                      )}
                    </div>
                    <p className="text-muted-foreground text-sm">
                      Or enter this key manually:{" "}
                      <code className="rounded bg-muted px-1 py-0.5 font-mono text-sm">
                        {secret}
                      </code>
                    </p>
                  </div>
                </Field>
              </FieldGroup>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="totp-code">Verification code</FieldLabel>
                  <Input
                    id="totp-code"
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    maxLength={6}
                    placeholder="000000"
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                    onKeyDown={(e) => e.key === "Enter" && handleVerify()}
                  />
                </Field>
              </FieldGroup>
              {error && <p className="text-destructive text-sm">{error}</p>}
            </FieldSet>
            <DialogFooter>
              <DialogClose>
                <Button variant="outline">Cancel</Button>
              </DialogClose>
              <Button onClick={handleVerify} disabled={isLoading}>
                {isLoading ? "Enabling..." : "Enable"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

interface RemoveTotpProps {
  onDisabled: () => void;
}

function RemoveTotp({ onDisabled }: RemoveTotpProps) {
  const { user } = useAuth();
  const [open, setOpen] = useState(false);
  const [password, setPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const resetState = () => {
    setPassword("");
    setError(null);
  };

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
    if (!open) {
      resetState();
    }
  };

  const handleDisable = async () => {
    if (!user) return;
    if (!password) {
      setError("Please enter your password");
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      await disableTOTP(user.uuid, password);
      setOpen(false);
      onDisabled();
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to disable two-factor authentication");
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger>
        <ListItem
          title="Remove two-factor authentication"
          leading={<LucideShieldOff />}
          trailing={<LucideChevronRight />}
        />
      </DialogTrigger>
      <DialogContent>
        <DialogTitle>Remove two-factor authentication</DialogTitle>
        <DialogDescription>
          Enter your password to disable two-factor authentication. This will
          make your account less secure.
        </DialogDescription>

        <FieldSet>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="disable-password">Password</FieldLabel>
              <Input
                id="disable-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleDisable()}
              />
            </Field>
          </FieldGroup>
          {error && <p className="text-destructive text-sm">{error}</p>}
        </FieldSet>

        <DialogFooter>
          <DialogClose>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button
            variant="destructive"
            onClick={handleDisable}
            disabled={isLoading}
          >
            {isLoading ? "Removing..." : "Remove"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
