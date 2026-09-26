"use client";

import React, { useState, useEffect, useMemo, useRef, useCallback, Suspense } from "react";
import { useInView } from "react-intersection-observer";
import { useRouter, useSearchParams } from "next/navigation";
import useSWR from "swr";
import useSWRInfinite from "swr/infinite";
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
  FaLayerGroup,
} from "react-icons/fa";

const SORT_OPTIONS = [
  { id: "trending", label: "Trending", icon: FaFire },
  { id: "date_desc", label: "Newest", icon: FaClock },
  { id: "views_desc", label: "Most Views", icon: FaEye },
  { id: "likes_desc", label: "Most Liked", icon: FaHeart },
  { id: "comments_desc", label: "Comments", icon: FaComments },
];

const UnifiedFeedContent = () => {
  const router = useRouter();
  const searchParams = useSearchParams();

  const { data: session, status } = useSession();
  const isAuthenticated = status === "authenticated";

  const PAGE_LIMIT = 8;

  const [sortBy, setSortBy] = useState(searchParams.get("sort") || "trending");
  const [activeCategory, setActiveCategory] = useState(
    searchParams.get("category") || "All"
  );
  const [videoType, setVideoType] = useState(
    searchParams.get("type") || "all" // "all", "standard", "short"
  );
  const [hasScrolled, setHasScrolled] = useState(false);

  // Only trigger auto-loading when user has deliberately scrolled
  useEffect(() => {
    const handleScroll = () => {
      if (window.scrollY > 40) {
        setHasScrolled(true);
      }
    };
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

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

  // Synchronize state when browser navigation (back/forward) alters URL params
  useEffect(() => {
    const sort = searchParams.get("sort") || "trending";
    const cat = searchParams.get("category") || "All";
    const type = searchParams.get("type") || "all";

    if (sort !== sortBy || cat !== activeCategory || type !== videoType) {
      setSortBy(sort);
      setActiveCategory(cat);
      setVideoType(type);
      setHasScrolled(false);
    }
  }, [searchParams]);

  // SWR Infinite Key Generator: stops requesting when last page is reached
  const getKey = (pageIndex, previousPageData) => {
    if (previousPageData) {
      const vids = previousPageData.videos;
      if (!Array.isArray(vids) || vids.length === 0 || vids.length < PAGE_LIMIT) {
        return null;
      }
      if (
        previousPageData.totalPages &&
        previousPageData.currentPage >= previousPageData.totalPages
      ) {
        return null;
      }
    }

    let url = `/videos?sort=${sortBy}&page=${pageIndex + 1}&limit=${PAGE_LIMIT}&type=${videoType}`;
    if (activeCategory && activeCategory !== "All") {
      url += `&category=${encodeURIComponent(activeCategory)}`;
    }
    return url;
  };

  // SWR Infinite configured to prevent cascading re-fetches on scroll
  const { data, error, isLoading, isValidating, size, setSize, mutate } =
    useSWRInfinite(getKey, fetcher, {
      revalidateFirstPage: false,
      revalidateAll: false,
      revalidateOnFocus: false,
      revalidateIfStale: false,
      persistSize: false,
    });

  // Deduplicate videos across all pages
  const videos = useMemo(() => {
    if (!data) return [];
    const seen = new Set();
    const result = [];
    for (const page of data) {
      if (Array.isArray(page?.videos)) {
        for (const v of page.videos) {
          if (v && v._id && !seen.has(v._id)) {
            seen.add(v._id);
            result.push(v);
          }
        }
      }
    }
    return result;
  }, [data]);

  const totalVideos = data?.[0]?.totalVideos ?? videos.length;
  const lastPage = data?.[data.length - 1];

  const isEmpty = Boolean(data?.[0]?.videos && data[0].videos.length === 0);

  const isReachingEnd = Boolean(
    isEmpty ||
      (data &&
        (lastPage?.videos?.length === 0 ||
          (lastPage?.totalPages && lastPage?.currentPage >= lastPage?.totalPages) ||
          (lastPage?.videos && lastPage.videos.length < PAGE_LIMIT) ||
          (lastPage?.totalVideos !== undefined && videos.length >= lastPage.totalVideos)))
  );

  const isLoadingMore =
    isLoading ||
    isValidating ||
    Boolean(size > 0 && data && typeof data[size - 1] === "undefined");

  // In-flight and cooldown locks to break infinite request loops
  const isFetchingRef = useRef(false);
  const wasLoadingMoreRef = useRef(false);

  useEffect(() => {
    if (isLoadingMore) {
      isFetchingRef.current = true;
    } else {
      const timer = setTimeout(() => {
        isFetchingRef.current = false;
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [isLoadingMore]);

  // Load more handler
  const handleLoadMore = useCallback(() => {
    if (isReachingEnd || isLoadingMore || isFetchingRef.current) return;
    setHasScrolled(true);
    isFetchingRef.current = true;
    setSize((prev) => prev + 1);
  }, [isReachingEnd, isLoadingMore, setSize]);

  // Intersection observer for sentinel
  const { ref, inView } = useInView({
    threshold: 0,
    rootMargin: "200px",
  });

  // Infinite scroll trigger: ONLY trigger when sentinel is in view and user has scrolled.
  // Explicitly ignore effect runs caused merely by loading finishing.
  useEffect(() => {
    const justFinishedLoading = wasLoadingMoreRef.current && !isLoadingMore;
    wasLoadingMoreRef.current = isLoadingMore;

    if (justFinishedLoading) {
      return;
    }

    if (
      inView &&
      hasScrolled &&
      !isReachingEnd &&
      !isLoadingMore &&
      !isFetchingRef.current
    ) {
      handleLoadMore();
    }
  }, [inView, hasScrolled, isReachingEnd, isLoadingMore, handleLoadMore]);

  // Handlers - Reset pagination size to 1 when filters change to prevent bulk page refetches
  const handleCategoryClick = (category) => {
    const nextCat = activeCategory === category ? "All" : category;
    setActiveCategory(nextCat);
    setSize(1);
    setHasScrolled(false);
    isFetchingRef.current = false;
    updateUrlParams(nextCat, sortBy, videoType);
  };

  const handleSortChange = (newSort) => {
    setSortBy(newSort);
    setSize(1);
    setHasScrolled(false);
    isFetchingRef.current = false;
    updateUrlParams(activeCategory, newSort, videoType);
  };

  const handleTypeChange = (newType) => {
    setVideoType(newType);
    setSize(1);
    setHasScrolled(false);
    isFetchingRef.current = false;
    updateUrlParams(activeCategory, sortBy, newType);
  };

  const updateUrlParams = (cat, sort, type) => {
    const params = new URLSearchParams();
    if (cat && cat !== "All") params.set("category", cat);
    if (sort && sort !== "trending") params.set("sort", sort);
    if (type && type !== "all") params.set("type", type);
    const queryString = params.toString();
    const targetUrl = queryString ? `/?${queryString}` : "/";
    router.replace(targetUrl, { scroll: false });
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
          </div>
          <p className="text-xs sm:text-sm text-gray-500 dark:text-gray-400 mt-1">
            Browse trending, popular, and latest uploads in one unified place.
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

          {/* Quick Sort Tabs on larger screens, Dropdown on small */}
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
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 sm:gap-6">
        {isLoading && videos.length === 0
          ? Array.from({ length: 8 }).map((_, idx) => (
              <VideoCardSkeleton key={idx} />
            ))
          : videos.map((video, idx) => (
              <motion.div
                key={video._id}
                initial={{ opacity: 0, y: 15 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25, delay: (idx % 8) * 0.03 }}
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

      {/* 4. Infinite Scroll Sentinel & Load More Fallback */}
      <div
        ref={ref}
        className="py-10 flex flex-col items-center justify-center space-y-3"
      >
        {isLoadingMore && (
          <div className="flex items-center space-x-2 text-indigo-600 dark:text-indigo-400 font-semibold text-xs sm:text-sm">
            <div className="w-4 h-4 border-2 border-indigo-600 dark:border-indigo-400 border-t-transparent rounded-full animate-spin" />
            <span>Loading more videos...</span>
          </div>
        )}

        {!isReachingEnd && !isLoadingMore && videos.length > 0 && (
          <button
            type="button"
            onClick={handleLoadMore}
            className="px-6 py-2.5 bg-white dark:bg-slate-800 hover:bg-gray-100 dark:hover:bg-slate-700 text-gray-800 dark:text-gray-200 font-bold text-xs sm:text-sm rounded-xl border border-gray-200 dark:border-slate-700 shadow-sm hover:shadow transition-all duration-200 transform hover:-translate-y-0.5 active:translate-y-0 cursor-pointer"
          >
            Load More Videos
          </button>
        )}

        {isReachingEnd && videos.length > 0 && (
          <div className="text-center py-3 px-6 rounded-2xl bg-gray-50 dark:bg-slate-800/40 border border-gray-200/60 dark:border-slate-800 text-xs sm:text-sm font-medium text-gray-500 dark:text-gray-400">
            🎉 You've reached the end of the feed.
          </div>
        )}
      </div>
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
            {Array.from({ length: 8 }).map((_, i) => (
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
