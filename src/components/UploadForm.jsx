"use client";

import React, { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import * as yup from "yup";
import API from "@/lib/api";
import CATEGORIES from "@/constants/categories";
import { useVideoUpload } from "@/hooks/useVideoUpload";
import { toast } from 'react-toastify';
import PlaylistUploadForm from "./PlaylistUploadForm";

import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Progress } from "@/components/ui/progress";

const schema = yup.object().shape({
  title: yup.string().required("Title is required"),
  description: yup.string().optional(),
  category: yup.string().required("Category is required"),
  tags: yup.string(),
  visibility: yup.string().required("Visibility is required"),
  playlistId: yup.string(),
  videoUrl: yup.string().url("Must be a valid URL").when("$uploadType", {
    is: "remote",
    then: (schema) => schema.required("Video URL is required for remote upload"),
  }),
  isShort: yup.boolean(),
});

const UploadForm = () => {
  const [uploadType, setUploadType] = useState("direct");
  const [playlists, setPlaylists] = useState([]);
  
  const {
    isUploading, uploadProgress, uploadSpeed, eta,
    isPolling, remoteProgress, statusMessage,
    uploadDirect, uploadRemote
  } = useVideoUpload();

  const { register, handleSubmit, formState: { errors }, watch, setValue } = useForm({
    resolver: yupResolver(schema),
    defaultValues: {
      title: "",
      description: "",
      category: "Other",
      tags: "",
      visibility: "public",
      playlistId: "",
      videoUrl: "",
      isShort: false
    },
    context: { uploadType }
  });

  const thumbnailFileList = watch("thumbnail");
  const videoFileList = watch("videoFile");

  useEffect(() => {
    const fetchPlaylists = async () => {
        try {
            const response = await API.get('/playlists/my-playlists');
            setPlaylists(response.data);
        } catch (error) {
            console.error("Could not fetch playlists", error);
        }
    };
    fetchPlaylists();
  }, []);

  const onSubmit = async (data) => {
    const thumbnailFile = thumbnailFileList?.[0];
    
    if (uploadType === "direct") {
      const videoFile = videoFileList?.[0];
      if (!videoFile) {
        toast.error("Please select a video file.");
        return;
      }
      await uploadDirect(videoFile, thumbnailFile, data);
    } else {
      if (!data.videoUrl) {
        toast.error("Please provide a video URL.");
        return;
      }
      await uploadRemote(data.videoUrl, thumbnailFile, data);
    }
  };

  const isBusy = isUploading || isPolling;

  return (
    <div className="bg-background flex items-center justify-center min-h-screen py-12 transition-colors duration-300">
      <Card className={`p-6 sm:p-8 rounded-2xl shadow-xl w-full ${uploadType === 'playlist' ? 'max-w-3xl' : 'max-w-2xl'} border-border transition-all duration-300`}>
        <h2 className="text-2xl font-bold mb-6 text-center text-foreground tracking-tight">
          {uploadType === 'playlist' ? 'Playlist Video Upload 📑' : 'Upload a New Video 🎬'}
        </h2>
        
        <div className="flex justify-center mb-6 rounded-xl p-1 bg-muted shadow-inner">
            <Button
                variant={uploadType === "direct" ? "default" : "ghost"}
                type="button"
                onClick={() => setUploadType("direct")}
                disabled={isBusy}
                className="w-1/3 rounded-lg font-semibold text-xs sm:text-sm transition-all duration-300 h-10"
            >
                Direct Upload
            </Button>
            <Button
                variant={uploadType === "remote" ? "default" : "ghost"}
                type="button"
                onClick={() => setUploadType("remote")}
                disabled={isBusy}
                className="w-1/3 rounded-lg font-semibold text-xs sm:text-sm transition-all duration-300 h-10"
            >
                Remote URL
            </Button>
            <Button
                variant={uploadType === "playlist" ? "default" : "ghost"}
                type="button"
                onClick={() => setUploadType("playlist")}
                disabled={isBusy}
                className="w-1/3 rounded-lg font-semibold text-xs sm:text-sm transition-all duration-300 h-10"
            >
                Playlist Upload
            </Button>
        </div>

        {uploadType === "playlist" ? (
          <PlaylistUploadForm
            playlists={playlists}
            onPlaylistCreated={(newPl) => setPlaylists((prev) => [...prev, newPl])}
          />
        ) : (

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            <div className="space-y-1.5">
                <Label htmlFor="title">Title</Label>
                <Input id="title" {...register("title")} type="text" placeholder="My Awesome Video" />
                {errors.title && <p className="text-destructive text-xs">{errors.title.message}</p>}
            </div>

            <div className="space-y-1.5">
                <Label htmlFor="tags">Tags</Label>
                <Input id="tags" {...register("tags")} type="text" placeholder="e.g., gaming, react, tutorial" />
                <p className="text-xs text-muted-foreground">Separate tags with a comma.</p>
            </div>

            <div className="space-y-1.5">
                <Label htmlFor="description">
                  Description <span className="text-xs font-normal text-muted-foreground">(Optional)</span>
                </Label>
                <Textarea id="description" {...register("description")} placeholder="A short description of your video..." className="h-24 resize-none" />
                {errors.description && <p className="text-destructive text-xs">{errors.description.message}</p>}
            </div>

            <div className="space-y-1.5">
                <Label htmlFor="thumbnail">Custom Thumbnail <span className="text-xs font-normal text-muted-foreground">(Optional)</span></Label>
                <Input id="thumbnail" {...register("thumbnail")} type="file" accept="image/*" className="cursor-pointer file:text-primary file:bg-primary/10 file:px-4 file:py-1 file:rounded-full file:border-none file:font-medium hover:file:bg-primary/20 file:mr-4 transition-colors" />
            </div>
            
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                 <div className="space-y-1.5">
                    <Label htmlFor="category">Category</Label>
                    <select id="category" {...register("category")} className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50">
                        {CATEGORIES.map(cat => (<option key={cat} value={cat}>{cat}</option>))}
                    </select>
                </div>
                <div className="space-y-1.5">
                    <Label htmlFor="playlistId">Add to Playlist (Optional)</Label>
                    <select id="playlistId" {...register("playlistId")} className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50">
                        <option value="">None</option>
                        {playlists.map(p => <option key={p._id} value={p._id}>{p.title}</option>)}
                    </select>
                </div>
                 <div className="space-y-1.5">
                    <Label htmlFor="visibility">Visibility</Label>
                    <select id="visibility" {...register("visibility")} className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50">
                        <option value="public">Public</option>
                        <option value="unlisted">Unlisted</option>
                        <option value="private">Private</option>
                    </select>
                </div>
            </div>
            <div className="flex items-start space-x-3 p-4 bg-muted/50 rounded-xl border border-border">
                <Checkbox 
                  id="isShort" 
                  onCheckedChange={(checked) => setValue("isShort", checked)}
                  className="mt-0.5"
                />
                <div className="grid gap-1.5 leading-none">
                  <Label htmlFor="isShort" className="font-bold cursor-pointer">
                    Upload as a Short (Vertical Video)
                  </Label>
                  <p className="text-xs text-muted-foreground">
                    This will display the video in the dedicated Shorts feed.
                  </p>
                </div>
            </div>

            {uploadType === 'direct' ? (
                <div className="space-y-1.5">
                    <Label htmlFor="videoFile">Video File</Label>
                    <Input id="videoFile" {...register("videoFile")} type="file" accept="video/*" className="cursor-pointer file:text-primary file:bg-primary/10 file:px-4 file:py-1 file:rounded-full file:border-none file:font-medium hover:file:bg-primary/20 file:mr-4 transition-colors" />
                </div>
            ) : (
                <div className="space-y-1.5">
                    <Label htmlFor="videoUrl">Video URL</Label>
                    <Input id="videoUrl" {...register("videoUrl")} type="url" placeholder="http://example.com/video.mp4" />
                    {errors.videoUrl && <p className="text-destructive text-xs">{errors.videoUrl.message}</p>}
                </div>
            )}
            
            {isUploading && (
                <div className="space-y-2 animate-in fade-in duration-300">
                    <Progress value={uploadProgress} className="h-2.5" />
                    <div className="flex justify-between text-sm font-medium text-muted-foreground">
                        <span>{uploadProgress}%</span>
                        <span>{uploadSpeed}</span>
                        <span>{eta}</span>
                    </div>
                </div>
            )}

            {isPolling && (
                <div className="space-y-2 animate-in fade-in duration-300">
                    <Progress value={remoteProgress} className="h-2.5" />
                    <div className="flex justify-between text-sm font-medium text-muted-foreground">
                        <span>{remoteProgress}%</span>
                        <span className="truncate">{statusMessage}</span>
                    </div>
                </div>
            )}

            <Button type="submit" disabled={isBusy} className="w-full h-12 text-md font-bold rounded-xl">
                {isUploading ? `Uploading... (${uploadProgress}%)` : isPolling ? "Processing..." : "Upload Video"}
            </Button>
        </form>
        )}
      </Card>
    </div>
  );
};

export default UploadForm;