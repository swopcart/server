import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ListItem } from "@/components/list-item";
import { LucideChevronRight, LucideKeyRound } from "lucide-react";

export function ChangePassword() {
  return (
    <Dialog>
      <DialogTrigger>
        <ListItem
          title="Change password"
          leading={<LucideKeyRound />}
          trailing={<LucideChevronRight />}
        />
      </DialogTrigger>
      <DialogContent>
        <DialogTitle>Change password</DialogTitle>

        <FieldSet>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="current-password">
                Current password
              </FieldLabel>
              <Input id="current-password" />
            </Field>
          </FieldGroup>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="new-password">New password</FieldLabel>
              <Input id="new-password" />
            </Field>
            <Field>
              <FieldLabel htmlFor="confirm-password">
                Confirm password
              </FieldLabel>
              <Input id="confirm-password" />
            </Field>
          </FieldGroup>
        </FieldSet>
        <DialogFooter>
          <DialogClose>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button>Change password</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
