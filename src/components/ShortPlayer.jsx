"use client";

import React, { useRef, useEffect, useState } from "react";
import { useInView } from "react-intersection-observer";
import Link from "next/link";
import Image from "next/image";
import {
  FaHeart,
  FaCommentDots,
  FaShare,
  FaPlay,
  FaVolumeMute,
  FaVolumeUp,
} from "react-icons/fa";
import { getAvatarUrl } from "@/lib/identicon";
import { useSession } from "next-auth/react";
import { toast } from "react-toastify";
import API from "@/lib/api";

const ShortPlayer = ({ video }) => {
  const videoRef = useRef(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [isLiked, setIsLiked] = useState(Boolean(video.isLiked));
  const [likesCount, setLikesCount] = useState(video.likesCount || 0);
  const [hasError, setHasError] = useState(false);

  const { data: session } = useSession();
  const { ref, inView } = useInView({
    threshold: 0.6,
  });

  useEffect(() => {
    const vid = videoRef.current;
    if (!vid) return;

    if (inView) {
      const playPromise = vid.play();
      if (playPromise !== undefined) {
        playPromise
          .then(() => {
            setIsPlaying(true);
          })
          .catch(() => {
            // Browser blocked unmuted autoplay: fall back to muted autoplay
            vid.muted = true;
            setIsMuted(true);
            vid
              .play()
              .then(() => setIsPlaying(true))
              .catch(() => setIsPlaying(false));
          });
      }
    } else {
      vid.pause();
      setIsPlaying(false);
    }
  }, [inView]);

  const togglePlay = () => {
    const vid = videoRef.current;
    if (!vid) return;

    if (isPlaying) {
      vid.pause();
      setIsPlaying(false);
    } else {
      vid
        .play()
        .then(() => setIsPlaying(true))
        .catch(() => setIsPlaying(false));
    }
  };

  const toggleMute = (e) => {
    e.stopPropagation();
    if (videoRef.current) {
      const nextMuted = !isMuted;
      videoRef.current.muted = nextMuted;
      setIsMuted(nextMuted);
    }
  };

  const handleLike = async (e) => {
    e.stopPropagation();
    if (!session?.user) {
      toast.warn("Please log in to like this short");
      return;
    }

    const nextLiked = !isLiked;
    setIsLiked(nextLiked);
    setLikesCount((prev) => (nextLiked ? prev + 1 : Math.max(0, prev - 1)));

    try {
      await API.post(`/videos/${video._id}/like`);
    } catch (err) {
      setIsLiked(!nextLiked);
      setLikesCount((prev) => (nextLiked ? Math.max(0, prev - 1) : prev + 1));
      toast.error("Failed to update like");
    }
  };

  const handleShare = async (e) => {
    e.stopPropagation();
    const url = `${window.location.origin}/video/${video._id}`;
    if (navigator.share) {
      try {
        await navigator.share({
          title: video.title,
          url,
        });
      } catch (err) {
        // User cancelled share
      }
    } else {
      await navigator.clipboard.writeText(url);
      toast.success("Link copied to clipboard!");
    }
  };

  const profileUsername = video.uploader?.username || "user";

  return (
    <div
      ref={ref}
      className="relative w-full max-w-[450px] h-[92%] max-h-[820px] bg-black rounded-2xl overflow-hidden mx-auto shadow-2xl flex items-center justify-center border border-white/10 group select-none"
    >
      {/* Video Element */}
      <video
        ref={videoRef}
        src={`/api/videos/stream/${video.fileId}`}
        className="w-full h-full object-cover cursor-pointer"
        loop
        playsInline
        preload="metadata"
        onClick={togglePlay}
        onError={() => setHasError(true)}
      />

      {/* Error Fallback */}
      {hasError && (
        <div className="absolute inset-0 flex flex-col items-center justify-center bg-slate-900/90 p-6 text-center text-white z-20">
          <p className="text-sm font-semibold mb-3">Unable to stream this short directly.</p>
          <Link
            href={`/video/${video._id}`}
            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-sm font-medium transition-all"
          >
            Watch on Video Page
          </Link>
        </div>
      )}

      {/* Play/Pause Overlay indicator */}
      {!isPlaying && !hasError && (
        <div
          onClick={togglePlay}
          className="absolute inset-0 flex items-center justify-center cursor-pointer bg-black/20"
        >
          <div className="bg-black/50 p-5 rounded-full backdrop-blur-md text-white hover:scale-110 transition-transform">
            <FaPlay className="text-3xl ml-1" />
          </div>
        </div>
      )}

      {/* Mute/Unmute Button (Top Right) */}
      <button
        onClick={toggleMute}
        aria-label={isMuted ? "Unmute" : "Mute"}
        className="absolute top-4 right-4 z-20 p-3 rounded-full bg-black/50 hover:bg-black/70 text-white backdrop-blur-md transition-all hover:scale-110"
      >
        {isMuted ? <FaVolumeMute size={18} /> : <FaVolumeUp size={18} />}
      </button>

      {/* Bottom Info Overlay */}
      <div className="absolute bottom-0 left-0 right-0 p-5 bg-gradient-to-t from-black/90 via-black/40 to-transparent flex justify-between items-end z-10 pointer-events-none">
        <div className="flex-1 pr-14 text-white pointer-events-auto">
          <Link
            href={`/profile/${profileUsername}`}
            className="flex items-center gap-3 mb-2.5 group/uploader w-fit"
          >
            <div className="w-10 h-10 rounded-full overflow-hidden ring-2 ring-white/40 group-hover/uploader:ring-white transition-all flex-shrink-0">
              <Image
                src={getAvatarUrl(profileUsername, video.uploader?.avatar)}
                alt={profileUsername}
                width={40}
                height={40}
                unoptimized
                className="object-cover w-full h-full"
              />
            </div>
            <span className="font-bold text-sm tracking-tight hover:underline">
              @{profileUsername}
            </span>
          </Link>
          <h3 className="font-semibold text-sm sm:text-base line-clamp-2 leading-snug drop-shadow-md">
            {video.title}
          </h3>
          {video.description && (
            <p className="text-xs sm:text-sm text-gray-300 line-clamp-1 mt-1 opacity-90 drop-shadow">
              {video.description}
            </p>
          )}
        </div>
      </div>

      {/* Right Side Action Buttons */}
      <div className="absolute bottom-8 right-3.5 flex flex-col gap-5 items-center z-10">
        {/* Like */}
        <button
          onClick={handleLike}
          className="flex flex-col items-center gap-1 group/btn focus:outline-none"
        >
          <div
            className={`p-3.5 rounded-full backdrop-blur-md transition-all group-hover/btn:scale-110 ${
              isLiked
                ? "bg-rose-600/90 text-white shadow-lg shadow-rose-600/30"
                : "bg-black/50 text-white hover:bg-black/70"
            }`}
          >
            <FaHeart
              className={`text-2xl transition-colors ${
                isLiked ? "text-white" : "group-hover/btn:text-rose-400"
              }`}
            />
          </div>
          <span className="text-white text-xs font-bold drop-shadow">
            {likesCount}
          </span>
        </button>

        {/* Comments */}
        <Link
          href={`/video/${video._id}`}
          className="flex flex-col items-center gap-1 group/btn focus:outline-none"
          title="View Comments"
        >
          <div className="p-3.5 rounded-full bg-black/50 hover:bg-black/70 text-white backdrop-blur-md transition-all group-hover/btn:scale-110">
            <FaCommentDots className="text-2xl" />
          </div>
          <span className="text-white text-xs font-bold drop-shadow">
            {video.commentCount || 0}
          </span>
        </Link>

        {/* Share */}
        <button
          onClick={handleShare}
          className="flex flex-col items-center gap-1 group/btn focus:outline-none"
          title="Share"
        >
          <div className="p-3.5 rounded-full bg-black/50 hover:bg-black/70 text-white backdrop-blur-md transition-all group-hover/btn:scale-110">
            <FaShare className="text-2xl" />
          </div>
          <span className="text-white text-xs font-bold drop-shadow">Share</span>
        </button>
      </div>
    </div>
  );
};

export default ShortPlayer;
