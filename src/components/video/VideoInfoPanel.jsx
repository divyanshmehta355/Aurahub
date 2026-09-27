"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import Image from "next/image";
import { getAvatarUrl } from "@/lib/identicon";
import { useSession } from "next-auth/react";
import { FcLike } from "react-icons/fc";
import { FaClock, FaCheck } from "react-icons/fa";
import { toast } from "react-toastify";
import API from "@/lib/api";
import { useThrottle } from "@/hooks/useThrottle";
import EditVideoModal from "@/components/EditVideoModal";
import SaveToPlaylistModal from "@/components/SaveToPlaylistModal";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";

const VideoInfoPanel = ({ initialVideo }) => {
  const { data: session, status } = useSession();
  const isAuthenticated = status === "authenticated";
  const user = session?.user;
  const router = useRouter();

  const [video, setVideo] = useState(initialVideo);
  const [inWatchLater, setInWatchLater] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showSaveModal, setShowSaveModal] = useState(false);

  useEffect(() => {
    setVideo(initialVideo);
  }, [initialVideo]);

  useEffect(() => {
    const checkWatchLaterStatus = async () => {
      if (isAuthenticated && video?._id) {
        try {
          const res = await API.get("/user/watch-later");
          const isSaved = res.data.some((item) => item.videoId._id === video._id);
          setInWatchLater(isSaved);
        } catch (error) {
          console.error("Failed to check watch later status");
        }
      }
    };
    checkWatchLaterStatus();
  }, [isAuthenticated, video?._id]);

  const handleLike = async () => {
    if (!isAuthenticated) {
      toast.warn("Please log in to like a video.");
      return;
    }
    try {
      setVideo((prev) =>
        prev
          ? {
              ...prev,
              isLiked: !prev.isLiked,
              likesCount: prev.isLiked
                ? prev.likesCount - 1
                : prev.likesCount + 1,
            }
          : null
      );
      await API.post(`/videos/${video._id}/like`);
    } catch (err) {
      toast.error("An error occurred while liking the video.");
      setVideo((prev) =>
        prev
          ? {
              ...prev,
              isLiked: !prev.isLiked,
              likesCount: prev.isLiked
                ? prev.likesCount + 1
                : prev.likesCount - 1,
            }
          : null
      );
    }
  };

  const throttledLikeHandler = useThrottle(handleLike, 2000);

  const handleToggleWatchLater = async () => {
    if (!isAuthenticated) {
      toast.warn("Please log in to save videos.");
      return;
    }
    try {
      if (inWatchLater) {
        await API.delete(`/user/watch-later?videoId=${video._id}`);
        toast.success("Removed from Watch Later");
      } else {
        await API.post("/user/watch-later", { videoId: video._id });
        toast.success("Added to Watch Later");
      }
      setInWatchLater(!inWatchLater);
    } catch (error) {
      toast.error("An error occurred.");
    }
  };

  const handleSaveEdits = async (data) => {
    try {
      const response = await API.put(`/videos/${video._id}`, data);
      setVideo((prev) =>
        prev
          ? {
              ...prev,
              title: response.data.title,
              description: response.data.description,
            }
          : null
      );
      setShowEditModal(false);
      toast.success("Video updated successfully!");
    } catch (err) {
      toast.error("Failed to update video.");
    }
  };

  const handleDelete = async () => {
    if (
      window.confirm(
        "Are you sure you want to delete this video? This action cannot be undone."
      )
    ) {
      try {
        await API.delete(`/videos/${video._id}`);
        toast.success("Video deleted successfully!");
        router.push("/");
      } catch (err) {
        toast.error("Failed to delete video.");
      }
    }
  };

  if (!video) return null;

  return (
    <>
      {showEditModal && (
        <EditVideoModal
          video={video}
          onSave={handleSaveEdits}
          onCancel={() => setShowEditModal(false)}
        />
      )}
      {showSaveModal && (
        <SaveToPlaylistModal
          videoId={video._id}
          onClose={() => setShowSaveModal(false)}
        />
      )}

      <motion.div 
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="mt-6 bg-card p-6 md:p-8 rounded-2xl shadow-xl border border-border transition-colors duration-300"
      >
        <h1 className="text-3xl font-bold text-foreground">{video.title}</h1>
        <div className="flex flex-col sm:flex-row justify-between sm:items-center mt-4 gap-4">
          <div className="text-sm font-medium text-muted-foreground">
            <span>{video.views} views</span>
            <span className="mx-2">•</span>
            <span>{new Date(video.createdAt).toLocaleDateString()}</span>
          </div>
          <div className="flex items-center space-x-2">
            <Button
              variant={video.isLiked ? "default" : "secondary"}
              onClick={throttledLikeHandler}
              className="rounded-xl px-5 h-10 font-semibold"
            >
              <FcLike className="mr-2" />
              <span>{video.likesCount}</span>
            </Button>
            {isAuthenticated && (
              <Button
                variant="secondary"
                onClick={() => setShowSaveModal(true)}
                className="rounded-xl px-5 h-10 font-semibold"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-4 w-4 mr-2"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path d="M5 4a2 2 0 012-2h6a2 2 0 012 2v14l-5-2.5L5 18V4z" />
                </svg>
                <span>Save</span>
              </Button>
            )}
            {isAuthenticated && (
              <Button
                variant="secondary"
                onClick={handleToggleWatchLater}
                className="rounded-xl px-5 h-10 font-semibold"
              >
                {inWatchLater ? (
                  <>
                    <FaCheck className="text-emerald-500 mr-2" />
                    <span>Saved</span>
                  </>
                ) : (
                  <>
                    <FaClock className="text-muted-foreground mr-2" />
                    <span>Watch Later</span>
                  </>
                )}
              </Button>
            )}
          </div>
        </div>
        <div className="mt-6 pt-6 border-t border-border">
          <div className="flex justify-between items-center">
            <div className="flex items-center gap-3">
              <Link href={`/profile/${video.uploader?.username}`}>
                <Avatar className="h-10 w-10 ring-2 ring-transparent hover:ring-primary transition-all duration-300">
                  <AvatarImage src={getAvatarUrl(video.uploader?.username, video.uploader?.avatar)} alt={video.uploader?.username || "Uploader"} />
                  <AvatarFallback>{video.uploader?.username?.charAt(0) || "U"}</AvatarFallback>
                </Avatar>
              </Link>
              <p className="text-foreground font-medium">
                Uploaded by{" "}
                <Link
                  href={`/profile/${video.uploader?.username}`}
                  className="font-bold hover:text-primary hover:underline transition-colors"
                >
                  {video.uploader?.username || "Unknown"}
                </Link>
              </p>
            </div>
            {user && user.id === video.uploader?._id && (
              <div className="flex space-x-3">
                <Button
                  variant="secondary"
                  onClick={() => setShowEditModal(true)}
                  className="rounded-xl font-semibold shadow-sm"
                >
                  Edit
                </Button>
                <Button
                  variant="destructive"
                  onClick={handleDelete}
                  className="rounded-xl font-semibold shadow-sm"
                >
                  Delete
                </Button>
              </div>
            )}
          </div>
          {video.description ? (
            <p className="mt-4 text-foreground whitespace-pre-wrap leading-relaxed">
              {video.description}
            </p>
          ) : null}
          {video.tags && video.tags.length > 0 && (
            <div className="mt-6 flex flex-wrap gap-2">
              {video.tags.map((tag) => (
                <span
                  key={tag}
                  className="px-3 py-1.5 bg-muted text-muted-foreground text-xs font-semibold rounded-lg"
                >
                  #{tag}
                </span>
              ))}
            </div>
          )}
        </div>
      </motion.div>
    </>
  );
};

export default VideoInfoPanel;
