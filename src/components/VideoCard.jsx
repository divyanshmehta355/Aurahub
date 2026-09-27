"use client";

import React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import VideoThumbnail from "./VideoThumbnail";
import { getAvatarUrl } from "@/lib/identicon";
import { FaClock, FaCheck } from "react-icons/fa";
import { useSession } from "next-auth/react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";

const VideoCard = ({ video, isSaved, onToggleWatchLater }) => {
  const router = useRouter();
  const { status } = useSession();
  const isAuthenticated = status === "authenticated";

  const handleAvatarClick = (e) => {
    e.stopPropagation();
    e.preventDefault();
    router.push(`/profile/${video.uploader?.username}`);
  };

  const handleWatchLaterClick = (e) => {
    e.stopPropagation();
    e.preventDefault();
    onToggleWatchLater();
  };

  return (
    <div className="block group cursor-pointer flex flex-col gap-3">
      <div className="relative w-full aspect-video rounded-xl overflow-hidden shadow-sm group-hover:shadow-lg transition-all duration-300">
        <Link href={`/video/${video._id}`} className="w-full h-full block transform group-hover:scale-105 transition-transform duration-500">
          <VideoThumbnail
            videoId={video._id}
            altText={video.title}
            title={video.title}
            category={video.category}
            thumbnailUrl={video.thumbnailUrl}
          />
        </Link>
        <div className="absolute inset-0 bg-foreground/0 group-hover:bg-foreground/10 transition-colors duration-300 pointer-events-none" />
        {isAuthenticated && (
          <Button
            variant="secondary"
            size="icon"
            onClick={handleWatchLaterClick}
            className="absolute top-2 right-2 h-8 w-8 bg-foreground/50 hover:bg-foreground/70 backdrop-blur-sm text-primary-foreground rounded-full transform hover:scale-110 transition-all duration-300 shadow-md border-none"
            title={isSaved ? "Remove from Watch Later" : "Watch Later"}
          >
            {isSaved ? <FaCheck className="text-emerald-400 h-3.5 w-3.5" /> : <FaClock className="h-3.5 w-3.5" />}
          </Button>
        )}
      </div>
      <div className="flex gap-3 px-1">
        <div
          className="flex-shrink-0 cursor-pointer"
          onClick={handleAvatarClick}
        >
          <Avatar className="h-9 w-9 ring-2 ring-transparent hover:ring-primary transition-all duration-300">
            <AvatarImage src={getAvatarUrl(video.uploader?.username, video.uploader?.avatar)} alt={video.uploader?.username || "Uploader"} />
            <AvatarFallback>{video.uploader?.username?.charAt(0) || "U"}</AvatarFallback>
          </Avatar>
        </div>

        <div className="flex flex-col w-0 flex-grow pt-0.5">
          <Link href={`/video/${video._id}`}>
            <h3 className="tracking-tight text-sm sm:text-base font-semibold text-foreground leading-snug line-clamp-2 transition-colors">
              {video.title}
            </h3>
          </Link>
          <p
            className="text-xs sm:text-sm text-muted-foreground mt-1 truncate hover:text-foreground transition-colors"
            onClick={handleAvatarClick}
          >
            {video.uploader?.username || "Unknown Uploader"}
          </p>
          <div className="flex items-center text-xs text-muted-foreground mt-0.5">
            <span>{video.views} views</span>
            <span className="mx-1.5 text-[10px]">•</span>
            <span>{new Date(video.createdAt).toLocaleDateString()}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default VideoCard;
