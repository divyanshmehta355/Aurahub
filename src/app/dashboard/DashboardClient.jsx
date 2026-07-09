"use client";

import React, { useState, useEffect, useRef } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useSession } from "next-auth/react";
import useSWR from "swr";
import { fetcher } from "@/lib/fetcher";
import API from "@/lib/api";
import AnalyticsChart from "@/components/AnalyticsChart";
import CreatorStatCards from "@/components/CreatorStatCards";
import TimeSeriesChart from "@/components/TimeSeriesChart";
import { useDebounce } from "@/hooks/useDebounce";
import VideoThumbnail from "@/components/VideoThumbnail";
import EditVideoModal from "@/components/EditVideoModal";
import ChangeThumbnailModal from "@/components/ChangeThumbnailModal";
import { toast } from "react-toastify";
import { MdOutlineAddPhotoAlternate } from "react-icons/md";
import { IoMdLink, IoIosLock, IoIosGlobe } from "react-icons/io";
import { motion } from "framer-motion";

const VisibilityDropdown = ({ video, onVisibilityChange }) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef(null);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const options = {
    public: {
      icon: <IoIosGlobe className="text-green-600" />,
      label: "Public",
    },
    unlisted: {
      icon: <IoMdLink className="text-yellow-600" />,
      label: "Unlisted",
    },
    private: { icon: <IoIosLock className="text-red-600" />, label: "Private" },
  };

  const selectedOption = options[video.visibility] || options.public;

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center text-sm rounded-xl border border-gray-200 dark:border-slate-700 px-3 py-1.5 bg-white dark:bg-slate-800 hover:bg-gray-50 dark:hover:bg-slate-700 text-gray-700 dark:text-gray-300 w-full justify-between transition-colors"
      >
        <div className="flex items-center">
          {selectedOption.icon}
          <span className="ml-2">{selectedOption.label}</span>
        </div>
        <svg
          className="h-4 w-4 text-gray-400"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>
      {isOpen && (
        <div className="absolute left-0 mt-2 w-full bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-gray-100 dark:border-slate-700 z-10 overflow-hidden">
          {Object.entries(options).map(([key, { icon, label }]) => (
            <button
              key={key}
              onClick={() => {
                onVisibilityChange(video._id, key);
                setIsOpen(false);
              }}
              className="w-full text-left px-4 py-2.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-slate-700/50 flex items-center transition-colors"
            >
              {icon} <span className="ml-2">{label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

const DashboardClient = () => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [searchTerm, setSearchTerm] = useState(searchParams.get("q") || "");
  const [editingVideo, setEditingVideo] = useState(null);
  const [changingThumbnailVideo, setChangingThumbnailVideo] = useState(null);

  const debouncedSearchTerm = useDebounce(searchTerm, 300);

  const { data: session, status } = useSession();

  const { data: videos, error, mutate, isLoading } = useSWR(
    status === "authenticated" 
      ? `/creator/dashboard${debouncedSearchTerm ? `?q=${encodeURIComponent(debouncedSearchTerm)}` : ""}` 
      : null,
    fetcher
  );

  const { data: analytics, error: analyticsError, isLoading: analyticsLoading } = useSWR(
    status === "authenticated" ? `/creator/analytics` : null,
    fetcher
  );

  useEffect(() => {
    if (status === "unauthenticated") {
      router.push("/login");
    }
  }, [status, router]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (searchTerm) {
      params.set("q", searchTerm);
    } else {
      params.delete("q");
    }
    router.replace(`/dashboard?${params.toString()}`);
  }, [searchTerm, router]);

  const handleClearSearch = () => setSearchTerm("");

  const handleDelete = async (videoId) => {
    if (window.confirm("Are you sure you want to delete this video?")) {
      try {
        await API.delete(`/videos/${videoId}`);
        mutate(videos.filter((v) => v._id !== videoId), false);
        toast.success("Video deleted successfully!");
      } catch (err) {
        toast.error("Failed to delete video.");
      }
    }
  };

  const handleSaveEdits = async (data) => {
    try {
      const response = await API.put(`/videos/${editingVideo._id}`, data);
      mutate(videos.map((v) => (v._id === editingVideo._id ? response.data : v)), false);
      setEditingVideo(null);
      toast.success("Video updated successfully!");
    } catch (err) {
      toast.error("Failed to update video.");
    }
  };

  const handleVisibilityChange = async (videoId, newVisibility) => {
    try {
      mutate(
        videos.map((v) =>
          v._id === videoId ? { ...v, visibility: newVisibility } : v
        ),
        false
      );
      await API.put(`/videos/${videoId}`, { visibility: newVisibility });
      toast.success("Visibility updated!");
    } catch (err) {
      toast.error("Failed to update visibility.");
    }
  };

  const handleThumbnailSave = (videoId, newThumbnailUrl) => {
    mutate(
      videos.map((v) =>
        v._id === videoId ? { ...v, thumbnailUrl: newThumbnailUrl } : v
      ),
      false
    );
    setChangingThumbnailVideo(null);
  };

  if (status === "loading" || isLoading || analyticsLoading) {
    return (
      <main className="container mx-auto px-6 py-8 animate-pulse">
        <div className="h-10 bg-gray-300 rounded w-1/3 mb-6"></div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          {[1, 2, 3, 4].map(i => <div key={i} className="h-32 bg-gray-300 rounded-2xl"></div>)}
        </div>
        <div className="mb-8 bg-white p-6 rounded-lg shadow-md">
          <div className="h-8 bg-gray-300 rounded w-1/4 mb-4"></div>
          <div className="h-80 bg-gray-300 rounded"></div>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-md">
          <div className="h-8 bg-gray-300 rounded w-1/4 mb-4"></div>
          <div className="h-40 bg-gray-300 rounded"></div>
        </div>
      </main>
    );
  }

  return (
    <>
      {editingVideo && (
        <EditVideoModal
          video={editingVideo}
          onSave={handleSaveEdits}
          onCancel={() => setEditingVideo(null)}
        />
      )}
      {changingThumbnailVideo && (
        <ChangeThumbnailModal
          video={changingThumbnailVideo}
          onSave={handleThumbnailSave}
          onCancel={() => setChangingThumbnailVideo(null)}
        />
      )}

      <motion.main 
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
        className="container mx-auto px-6 py-8"
      >
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-6">
          Creator Dashboard
        </h1>

        <div className="mb-8 bg-white dark:bg-slate-900 p-6 md:p-8 rounded-2xl shadow-xl border border-gray-100 dark:border-slate-800 transition-colors duration-300">
          <div className="flex flex-col sm:flex-row justify-between sm:items-center mb-6 gap-4">
            <h2 className="text-xl font-bold text-gray-900 dark:text-white">Analytics Overview</h2>
            <div className="relative w-full sm:w-1/3">
              <input
                type="text"
                placeholder="Search to filter..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-4 pr-10 py-2.5 border border-gray-200 dark:border-slate-700 rounded-xl bg-white dark:bg-slate-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 transition-all"
              />
              {searchTerm && (
                <button
                  onClick={handleClearSearch}
                  className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-5 w-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              )}
            </div>
          </div>

          {analyticsError ? (
            <div className="text-center text-rose-500 dark:text-rose-400 p-8 font-medium">Failed to load analytics data.</div>
          ) : analytics ? (
            <>
              <CreatorStatCards stats={analytics.lifetimeStats} />
              <div className="mb-8">
                <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-200 mb-4">30-Day Channel Growth</h3>
                <TimeSeriesChart timeSeries={analytics.timeSeries} />
              </div>
            </>
          ) : null}

          {error ? (
            <div className="text-center text-rose-500 dark:text-rose-400 p-8 font-medium">Failed to load video data.</div>
          ) : videos && videos.length > 0 ? (
            <div>
              <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-200 mb-4 mt-8 pt-8 border-t border-gray-100 dark:border-slate-800">Video Performance Comparison</h3>
              <AnalyticsChart videos={videos} />
            </div>
          ) : (
            <div className="text-center text-gray-500 dark:text-gray-400 p-8 font-medium">
              {searchTerm
                ? "No videos found matching your search."
                : "No data to display."}
            </div>
          )}
        </div>

        <div className="bg-white dark:bg-slate-900 p-6 md:p-8 rounded-2xl shadow-xl border border-gray-100 dark:border-slate-800 transition-colors duration-300">
          <div className="flex flex-col sm:flex-row justify-between sm:items-center mb-6 gap-4">
            <h2 className="text-xl font-bold text-gray-900 dark:text-white">Your Videos</h2>
            <div className="relative w-full sm:w-1/3">
              <input
                type="text"
                placeholder="Search your videos..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-4 pr-10 py-2.5 border border-gray-200 dark:border-slate-700 rounded-xl bg-white dark:bg-slate-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 transition-all"
              />
              {searchTerm && (
                <button
                  onClick={handleClearSearch}
                  className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-5 w-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              )}
            </div>
          </div>

          <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-slate-700">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-slate-700">
              <thead className="bg-gray-50 dark:bg-slate-800/50">
                <tr>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Thumbnail</th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Video Title</th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Visibility</th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Views</th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Likes</th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Comments</th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-slate-900 divide-y divide-gray-200 dark:divide-slate-700">
                {videos.length > 0 ? (
                  videos.map((video) => (
                    <tr key={video._id}>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="relative w-24 h-14 group">
                          <VideoThumbnail
                            videoId={video._id}
                            altText={video.title}
                          />
                          <button
                            onClick={() => setChangingThumbnailVideo(video)}
                            className="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center text-white font-semibold opacity-0 group-hover:opacity-100 transition-opacity rounded-lg"
                          >
                            <MdOutlineAddPhotoAlternate />
                          </button>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link href={`/video/${video._id}`} className="font-semibold text-gray-900 dark:text-white hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors">{video.title}</Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <VisibilityDropdown
                          video={video}
                          onVisibilityChange={handleVisibilityChange}
                        />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-gray-600 dark:text-gray-400 font-medium">{video.views}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-gray-600 dark:text-gray-400 font-medium">{video.likesCount}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-gray-600 dark:text-gray-400 font-medium">{video.commentCount}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                        <div className="flex items-center space-x-3">
                          <button onClick={() => setEditingVideo(video)} className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 transition-colors">
                            Edit
                          </button>
                          <button onClick={() => handleDelete(video._id)} className="text-rose-600 dark:text-rose-400 hover:text-rose-800 dark:hover:text-rose-300 transition-colors">
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan="7"
                      className="px-6 py-12 text-center text-gray-500 dark:text-gray-400"
                    >
                      {searchTerm
                        ? "No videos found..."
                        : "You haven't uploaded any videos yet."}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </motion.main>
    </>
  );
};

export default DashboardClient;
