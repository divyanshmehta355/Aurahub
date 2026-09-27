"use client";

import React, { useState, useRef } from "react";
import { toast } from "react-toastify";
import API from "@/lib/api";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const ChangeThumbnailModal = ({ video, onSave, onCancel }) => {
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef(null);
  const [previewUrl, setPreviewUrl] = useState(null);

  const handleFileChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      const reader = new FileReader();
      reader.onloadend = () => {
        setPreviewUrl(reader.result);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleSave = async () => {
    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      toast.error("Please select a new thumbnail image.");
      return;
    }

    setIsUploading(true);
    const formData = new FormData();
    formData.append("thumbnailFile", file);

    try {
      const response = await API.post(
        `/videos/${video._id}/update-thumbnail`,
        formData
      );
      onSave(video._id, response.data.thumbnailUrl);
      toast.success("Thumbnail updated successfully!");
    } catch (error) {
      toast.error("Failed to update thumbnail.");
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <Dialog open={true} onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Change Thumbnail</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <Input
            type="file"
            ref={fileInputRef}
            onChange={handleFileChange}
            accept="image/*"
            className="cursor-pointer file:text-primary file:bg-primary/10 file:px-4 file:py-1 file:rounded-full file:border-none file:font-medium hover:file:bg-primary/20 transition-colors"
          />
          {previewUrl && (
            <div className="mt-4 border border-border rounded-xl p-2 bg-muted">
              <img
                src={previewUrl}
                alt="New thumbnail preview"
                className="w-full h-auto rounded-lg shadow-sm"
              />
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onCancel} disabled={isUploading}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={isUploading}>
            {isUploading ? "Uploading..." : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ChangeThumbnailModal;
