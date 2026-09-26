"use client";

import React, { useEffect, useRef, useCallback } from "react";
import useSWRInfinite from "swr/infinite";
import { fetcher } from "@/lib/fetcher";
import SuggestedVideoCard from "@/components/SuggestedVideoCard";
import { useInView } from "react-intersection-observer";

const VideoRecommendations = ({ videoId, isAuthenticated }) => {
  const { ref, inView } = useInView({ threshold: 0.2 });

  const endpoint = isAuthenticated ? "/videos/recommendations" : "/videos/suggestions";

  const getKey = (pageIndex, previousPageData) => {
    // Reached the end
    if (previousPageData) {
      const vids = previousPageData.videos || (Array.isArray(previousPageData) ? previousPageData : null);
      if (!vids || vids.length === 0 || vids.length < 10) return null;
      if (previousPageData.totalPages && previousPageData.currentPage >= previousPageData.totalPages) {
        return null;
      }
    }

    // add the cursor to the API endpoint
    return `${endpoint}?page=${pageIndex + 1}&limit=10&exclude=${videoId}`;
  };

  const { data, error, isLoading, isValidating, size, setSize } = useSWRInfinite(
    getKey,
    fetcher,
    {
      revalidateFirstPage: false,
      revalidateAll: false,
      revalidateOnFocus: false,
      revalidateIfStale: false,
      persistSize: false,
    }
  );

  const videos = data 
    ? data.flatMap(page => page.videos || (Array.isArray(page) ? page : []))
    : [];

  const lastPage = data?.[data.length - 1];
  const lastPageVids = lastPage?.videos || (Array.isArray(lastPage) ? lastPage : []);
  const isEmpty = data?.[0]?.length === 0 || data?.[0]?.videos?.length === 0;
  const isReachingEnd =
    isEmpty ||
    Boolean(
      data &&
        (lastPageVids.length < 10 ||
          (lastPage?.totalPages && lastPage?.currentPage >= lastPage?.totalPages))
    );

  const isLoadingMore =
    isLoading ||
    isValidating ||
    Boolean(size > 0 && data && typeof data[size - 1] === "undefined");

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

  const handleLoadMore = useCallback(() => {
    if (isReachingEnd || isLoadingMore || isFetchingRef.current) return;
    isFetchingRef.current = true;
    setSize((prev) => prev + 1);
  }, [isReachingEnd, isLoadingMore, setSize]);

  useEffect(() => {
    const justFinishedLoading = wasLoadingMoreRef.current && !isLoadingMore;
    wasLoadingMoreRef.current = isLoadingMore;

    if (justFinishedLoading) return;

    if (inView && !isReachingEnd && !isLoadingMore && !isFetchingRef.current) {
      handleLoadMore();
    }
  }, [inView, isReachingEnd, isLoadingMore, handleLoadMore]);

  if (error) {
    return <div className="text-red-500">Failed to load recommendations.</div>;
  }

  return (
    <div className="sticky top-24">
      <h3 className="font-bold text-xl text-gray-900 dark:text-white mb-6">
        {isAuthenticated ? "Recommended For You" : "Up Next"}
      </h3>
      <div className="space-y-4 max-h-[80vh] overflow-y-auto pr-2 custom-scrollbar">
        {videos.map((sidebarVideo) => (
          <SuggestedVideoCard key={sidebarVideo._id} video={sidebarVideo} />
        ))}
        
        {isLoadingMore && (
          Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="flex space-x-3 animate-pulse">
              <div className="flex-shrink-0 w-40 h-24 bg-gray-200 dark:bg-slate-800 rounded-lg"></div>
              <div className="flex-1 space-y-3 py-1">
                <div className="h-4 bg-gray-200 dark:bg-slate-800 rounded w-full"></div>
                <div className="h-4 bg-gray-200 dark:bg-slate-800 rounded w-2/3"></div>
              </div>
            </div>
          ))
        )}

        <div ref={ref} className="h-10">
          {!isReachingEnd && !isLoadingMore && (
             <p className="text-center text-sm font-medium text-gray-500 dark:text-gray-400 mt-4 cursor-pointer" onClick={handleLoadMore}>
               Loading more...
             </p>
          )}
        </div>
      </div>
    </div>
  );
};

export default VideoRecommendations;
