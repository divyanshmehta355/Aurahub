"use client";

import React, { useState, useEffect, useMemo, useRef, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import useSWR from "swr";
import { fetcher } from "@/lib/fetcher";
import API from "@/lib/api";
import { motion, AnimatePresence } from "framer-motion";
import VideoCard from "@/components/VideoCard";
import VideoCardSkeleton from "@/components/VideoCardSkeleton";
import CATEGORIES from "@/constants/categories";
import { useSession } from "next-auth/react";
import { toast } from "react-toastify";
import {
  FaFire,
  FaClock,
  FaEye,
  FaHeart,
  FaComments,
  FaCompass,
  FaChevronLeft,
  FaChevronRight,
} from "react-icons/fa";

const SORT_OPTIONS = [
  { id: "trending", label: "Trending", icon: FaFire },
  { id: "date_desc", label: "Newest", icon: FaClock },
  { id: "views_desc", label: "Most Views", icon: FaEye },
  { id: "likes_desc", label: "Most Liked", icon: FaHeart },
  { id: "comments_desc", label: "Comments", icon: FaComments },
];

const PAGE_LIMIT = 12;

const UnifiedFeedContent = () => {
  const router = useRouter();
  const searchParams = useSearchParams();

  const { data: session, status } = useSession();
  const isAuthenticated = status === "authenticated";

  const gridRef = useRef(null);

  // Parse state from URL search params
  const [currentPage, setCurrentPage] = useState(() => {
    const p = parseInt(searchParams.get("page") || "1", 10);
    return isNaN(p) || p < 1 ? 1 : p;
  });
  const [sortBy, setSortBy] = useState(() => searchParams.get("sort") || "trending");
  const [activeCategory, setActiveCategory] = useState(
    () => searchParams.get("category") || "All"
  );
  const [videoType, setVideoType] = useState(
    () => searchParams.get("type") || "all" // "all", "standard", "short"
  );

  // Synchronize state when browser navigation (back/forward) alters URL params
  useEffect(() => {
    const p = parseInt(searchParams.get("page") || "1", 10);
    const validPage = isNaN(p) || p < 1 ? 1 : p;
    const sort = searchParams.get("sort") || "trending";
    const cat = searchParams.get("category") || "All";
    const type = searchParams.get("type") || "all";

    setCurrentPage(validPage);
    setSortBy(sort);
    setActiveCategory(cat);
    setVideoType(type);
  }, [searchParams]);

  // Watch Later Cache
  const { data: watchLaterData, mutate: mutateWatchLater } = useSWR(
    isAuthenticated ? "/user/watch-later" : null,
    fetcher
  );

  const watchLaterIds = useMemo(() => {
    if (!watchLaterData || !Array.isArray(watchLaterData)) return new Set();
    return new Set(
      watchLaterData
        .map((item) => item?.videoId?._id)
        .filter(Boolean)
    );
  }, [watchLaterData]);

  // Construct discrete page endpoint for traditional pagination
  const apiEndpoint = useMemo(() => {
    let url = `/videos?sort=${sortBy}&page=${currentPage}&limit=${PAGE_LIMIT}&type=${videoType}`;
    if (activeCategory && activeCategory !== "All") {
      url += `&category=${encodeURIComponent(activeCategory)}`;
    }
    return url;
  }, [sortBy, currentPage, videoType, activeCategory]);

  // Standard useSWR for discrete page fetching with keepPreviousData for smooth transitions
  const { data, error, isLoading, isValidating } = useSWR(apiEndpoint, fetcher, {
    keepPreviousData: true,
    revalidateOnFocus: false,
  });

  const videos = data?.videos || [];
  const totalPages = data?.totalPages || 1;
  const totalVideos = data?.totalVideos || 0;

  // Build pagination numbers array with ellipsis
  const paginationItems = useMemo(() => {
    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, i) => i + 1);
    }
    if (currentPage <= 4) {
      return [1, 2, 3, 4, 5, "...", totalPages];
    }
    if (currentPage >= totalPages - 3) {
      return [
        1,
        "...",
        totalPages - 4,
        totalPages - 3,
        totalPages - 2,
        totalPages - 1,
        totalPages,
      ];
    }
    return [
      1,
      "...",
      currentPage - 1,
      currentPage,
      currentPage + 1,
      "...",
      totalPages,
    ];
  }, [currentPage, totalPages]);

  // URL updater
  const updateUrlParams = (cat, sort, type, page = 1) => {
    const params = new URLSearchParams();
    if (cat && cat !== "All") params.set("category", cat);
    if (sort && sort !== "trending") params.set("sort", sort);
    if (type && type !== "all") params.set("type", type);
    if (page && page > 1) params.set("page", page.toString());
    const queryString = params.toString();
    const targetUrl = queryString ? `/?${queryString}` : "/";
    router.push(targetUrl, { scroll: false });
  };

  // Filter change handlers (always reset to page 1)
  const handleCategoryClick = (category) => {
    const nextCat = activeCategory === category ? "All" : category;
    setActiveCategory(nextCat);
    setCurrentPage(1);
    updateUrlParams(nextCat, sortBy, videoType, 1);
  };

  const handleSortChange = (newSort) => {
    setSortBy(newSort);
    setCurrentPage(1);
    updateUrlParams(activeCategory, newSort, videoType, 1);
  };

  const handleTypeChange = (newType) => {
    setVideoType(newType);
    setCurrentPage(1);
    updateUrlParams(activeCategory, sortBy, newType, 1);
  };

  const handlePageChange = (newPage) => {
    if (newPage < 1 || newPage > totalPages || newPage === currentPage) return;
    setCurrentPage(newPage);
    updateUrlParams(activeCategory, sortBy, videoType, newPage);

    // Smooth scroll back to the top of the video grid
    if (gridRef.current) {
      const topOffset = gridRef.current.getBoundingClientRect().top + window.scrollY - 80;
      window.scrollTo({ top: topOffset, behavior: "smooth" });
    } else {
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  const handleToggleWatchLater = async (videoId) => {
    if (!isAuthenticated) {
      toast.warn("Please log in to save videos.");
      return;
    }
    const isSaved = watchLaterIds.has(videoId);

    try {
      if (isSaved) {
        await API.delete(`/user/watch-later?videoId=${videoId}`);
        mutateWatchLater(
          (watchLaterData || []).filter((item) => item.videoId?._id !== videoId),
          false
        );
        toast.success("Removed from Watch Later");
      } else {
        await API.post("/user/watch-later", { videoId });
        mutateWatchLater();
        toast.success("Added to Watch Later");
      }
    } catch {
      toast.error("Could not update Watch Later.");
    }
  };

  const currentSortObj =
    SORT_OPTIONS.find((s) => s.id === sortBy) || SORT_OPTIONS[0];

  const startIndex = totalVideos > 0 ? (currentPage - 1) * PAGE_LIMIT + 1 : 0;
  const endIndex = Math.min(currentPage * PAGE_LIMIT, totalVideos);

  return (
    <main className="container mx-auto px-4 sm:px-6 py-6 sm:py-8">
      {/* 1. Category Filter Pills Carousel */}
      <div className="mb-6 flex items-center gap-2 overflow-x-auto pb-2 scrollbar-hide">
        <button
          onClick={() => handleCategoryClick("All")}
          className={`px-4 sm:px-5 py-2 rounded-full font-bold text-xs sm:text-sm whitespace-nowrap transition-all duration-300 shadow-sm ${
            activeCategory === "All"
              ? "bg-indigo-600 text-white shadow-md hover:bg-indigo-700"
              : "bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-slate-800 dark:text-gray-300 dark:hover:bg-slate-700"
          }`}
        >
          All
        </button>
        {CATEGORIES.map((cat) => (
          <button
            key={cat}
            onClick={() => handleCategoryClick(cat)}
            className={`px-4 sm:px-5 py-2 rounded-full font-bold text-xs sm:text-sm whitespace-nowrap transition-all duration-300 shadow-sm ${
              activeCategory === cat
                ? "bg-indigo-600 text-white shadow-md hover:bg-indigo-700"
                : "bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-slate-800 dark:text-gray-300 dark:hover:bg-slate-700"
            }`}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* 2. Unified Header & Control Toolbar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8 pb-4 border-b border-gray-100 dark:border-slate-800">
        <div>
          <div className="flex items-center space-x-2">
            <h1 className="text-2xl sm:text-3xl font-extrabold text-gray-900 dark:text-white tracking-tight">
              {activeCategory === "All"
                ? `${currentSortObj.label} Videos`
                : `${activeCategory} Videos`}
            </h1>
            {isValidating && !isLoading && (
              <div className="w-2 h-2 rounded-full bg-indigo-500 animate-ping" title="Refreshing..." />
            )}
          </div>
          <p className="text-xs sm:text-sm text-gray-500 dark:text-gray-400 mt-1">
            Browse trending, popular, and latest uploads with traditional page navigation.
          </p>
        </div>

        {/* Controls: Type Tabs & Sort Dropdown */}
        <div className="flex items-center gap-3 flex-wrap">
          {/* Format Type Selector */}
          <div className="flex rounded-xl p-1 bg-gray-100 dark:bg-slate-800/80 text-xs font-semibold">
            {[
              { id: "all", label: "All" },
              { id: "standard", label: "Videos" },
              { id: "short", label: "Shorts" },
            ].map((t) => (
              <button
                key={t.id}
                onClick={() => handleTypeChange(t.id)}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  videoType === t.id
                    ? "bg-white dark:bg-slate-700 text-indigo-600 dark:text-indigo-300 shadow-sm"
                    : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>

          {/* Quick Sort Dropdown */}
          <div className="flex items-center space-x-2">
            <label
              htmlFor="unified-sort"
              className="text-xs font-bold text-gray-500 dark:text-gray-400"
            >
              Sort:
            </label>
            <select
              id="unified-sort"
              value={sortBy}
              onChange={(e) => handleSortChange(e.target.value)}
              className="px-3 py-1.5 bg-white dark:bg-slate-800 border border-gray-200 dark:border-slate-700 rounded-xl text-xs sm:text-sm font-semibold text-gray-800 dark:text-gray-200 focus:ring-2 focus:ring-indigo-500 outline-none cursor-pointer shadow-sm"
            >
              {SORT_OPTIONS.map((opt) => (
                <option key={opt.id} value={opt.id}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Error State */}
      {error && (
        <div className="p-4 mb-6 rounded-xl bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 text-center text-sm font-medium">
          {error?.message || "Failed to load videos. Please try again."}
        </div>
      )}

      {/* 3. Responsive Video Grid */}
      <div
        ref={gridRef}
        className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 sm:gap-6 min-h-[400px]"
      >
        {isLoading && videos.length === 0
          ? Array.from({ length: PAGE_LIMIT }).map((_, idx) => (
              <VideoCardSkeleton key={idx} />
            ))
          : videos.map((video, idx) => (
              <motion.div
                key={video._id}
                initial={{ opacity: 0, y: 15 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.2, delay: (idx % 4) * 0.04 }}
              >
                <VideoCard
                  video={video}
                  isSaved={watchLaterIds.has(video._id)}
                  onToggleWatchLater={() => handleToggleWatchLater(video._id)}
                />
              </motion.div>
            ))}
      </div>

      {/* Empty State */}
      {!isLoading && videos.length === 0 && (
        <div className="text-center py-16 px-4 bg-white dark:bg-slate-900/40 rounded-2xl border border-gray-100 dark:border-slate-800 my-6 shadow-sm">
          <FaCompass size={40} className="mx-auto text-indigo-400 mb-3 opacity-80" />
          <h3 className="text-lg font-bold text-gray-900 dark:text-white">
            No videos found
          </h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-sm mx-auto">
            {activeCategory !== "All"
              ? `There are no videos in the "${activeCategory}" category yet.`
              : "No videos match your current filter."}
          </p>
          {activeCategory !== "All" && (
            <button
              onClick={() => handleCategoryClick("All")}
              className="mt-4 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-bold rounded-xl transition-colors shadow-sm"
            >
              Show All Categories
            </button>
          )}
        </div>
      )}

      {/* 4. Traditional Pagination Navigation Bar */}
      {totalPages > 1 && (
        <nav
          aria-label="Pagination Navigation"
          className="mt-10 pt-6 border-t border-gray-100 dark:border-slate-800/80 flex flex-col sm:flex-row items-center justify-between gap-4"
        >
          {/* Summary / Range text */}
          <div className="text-xs sm:text-sm font-medium text-gray-500 dark:text-gray-400 order-2 sm:order-1">
            Showing <span className="font-bold text-gray-900 dark:text-white">{startIndex}</span>–
            <span className="font-bold text-gray-900 dark:text-white">{endIndex}</span> of{" "}
            <span className="font-bold text-gray-900 dark:text-white">{totalVideos}</span> videos
          </div>

          {/* Pagination Controls */}
          <div className="flex items-center gap-1.5 sm:gap-2 order-1 sm:order-2">
            {/* Previous Page Button */}
            <button
              onClick={() => handlePageChange(currentPage - 1)}
              disabled={currentPage <= 1 || isLoading}
              aria-label="Previous Page"
              className="px-3.5 py-2 rounded-xl text-xs sm:text-sm font-semibold flex items-center gap-1.5 transition-all duration-200 disabled:opacity-30 disabled:cursor-not-allowed bg-gray-100 hover:bg-gray-200 text-gray-700 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-gray-200 shadow-sm cursor-pointer"
            >
              <FaChevronLeft className="w-3 h-3" />
              <span>Prev</span>
            </button>

            {/* Page Number Buttons */}
            {paginationItems.map((item, idx) => {
              if (item === "...") {
                return (
                  <span
                    key={`ellipsis-${idx}`}
                    className="w-8 h-9 sm:h-10 flex items-center justify-center text-xs sm:text-sm text-gray-400 dark:text-gray-500 select-none"
                  >
                    …
                  </span>
                );
              }

              const isCurrent = item === currentPage;
              return (
                <button
                  key={item}
                  onClick={() => handlePageChange(item)}
                  disabled={isLoading}
                  aria-current={isCurrent ? "page" : undefined}
                  className={`min-w-[36px] sm:min-w-[40px] h-9 sm:h-10 px-2 rounded-xl text-xs sm:text-sm font-bold transition-all duration-200 cursor-pointer ${
                    isCurrent
                      ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/25 scale-105"
                      : "bg-gray-100 hover:bg-gray-200 text-gray-700 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-gray-300"
                  }`}
                >
                  {item}
                </button>
              );
            })}

            {/* Next Page Button */}
            <button
              onClick={() => handlePageChange(currentPage + 1)}
              disabled={currentPage >= totalPages || isLoading}
              aria-label="Next Page"
              className="px-3.5 py-2 rounded-xl text-xs sm:text-sm font-semibold flex items-center gap-1.5 transition-all duration-200 disabled:opacity-30 disabled:cursor-not-allowed bg-gray-100 hover:bg-gray-200 text-gray-700 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-gray-200 shadow-sm cursor-pointer"
            >
              <span>Next</span>
              <FaChevronRight className="w-3 h-3" />
            </button>
          </div>
        </nav>
      )}
    </main>
  );
};

const UnifiedFeedClient = () => {
  return (
    <Suspense
      fallback={
        <div className="container mx-auto px-4 sm:px-6 py-8">
          <div className="h-10 bg-gray-100 dark:bg-slate-800 rounded-full w-2/3 mb-8 animate-pulse" />
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
            {Array.from({ length: PAGE_LIMIT }).map((_, i) => (
              <VideoCardSkeleton key={i} />
            ))}
          </div>
        </div>
      }
    >
      <UnifiedFeedContent />
    </Suspense>
  );
};

export default UnifiedFeedClient;
