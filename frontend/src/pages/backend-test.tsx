import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { useState } from "react";

export function BackendTest() {
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<string | undefined>();

  const onPingClicked = () => {
    setPending(true);
    fetch("/api/v0/ping")
      .then((response) => {
        return response.json();
      })
      .then((body) => {
        setMessage(
          `Server responded with ${JSON.stringify(body, undefined, 2)}`,
        );
      })
      .catch((error) => {
        setMessage(`Failed to ping server: ${error}`);
      })
      .finally(() => setPending(false));
  };

  return (
    <div className="flex flex-col justify-center items-center min-h-screen">
      <Card className="w-[320px]">
        <CardHeader>
          <CardTitle>Test server connection</CardTitle>
        </CardHeader>
        <CardContent>
          {pending ? (
            <Spinner />
          ) : (
            (message ?? "No message received from server.")
          )}
        </CardContent>
        <CardFooter className="flex-col gap-2">
          <Button className="w-full" disabled={pending} onClick={onPingClicked}>
            Ping server
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}
