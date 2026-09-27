import React from "react";
import { useForm } from "react-hook-form";
import * as Yup from "yup";
import { yupResolver } from "@hookform/resolvers/yup";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";

const editSchema = Yup.object().shape({
  title: Yup.string().required("Title is required"),
  description: Yup.string().optional(),
});

const EditVideoModal = ({ video, onSave, onCancel }) => {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: yupResolver(editSchema),
    defaultValues: {
      title: video.title,
      description: video.description,
    },
  });

  return (
    <Dialog open={true} onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit Video Details</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSave)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="title">Title</Label>
            <Input
              id="title"
              type="text"
              {...register("title")}
              className={errors.title ? "border-destructive focus-visible:ring-destructive" : ""}
            />
            {errors.title && (
              <p className="text-xs text-destructive mt-1">
                {errors.title.message}
              </p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="description">
              Description <span className="text-xs text-muted-foreground font-normal">(Optional)</span>
            </Label>
            <Textarea
              id="description"
              rows={5}
              {...register("description")}
              className={`resize-none ${errors.description ? "border-destructive focus-visible:ring-destructive" : ""}`}
            />
            {errors.description && (
              <p className="text-xs text-destructive mt-1">
                {errors.description.message}
              </p>
            )}
          </div>
          <DialogFooter className="mt-8">
            <Button variant="ghost" type="button" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit">
              Save Changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};

export default EditVideoModal;
