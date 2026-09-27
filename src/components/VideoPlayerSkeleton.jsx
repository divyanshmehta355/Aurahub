import React from "react";
import { Skeleton } from "@/components/ui/skeleton";

const VideoPlayerSkeleton = () => {
  return (
    <main className="max-w-screen-2xl mx-auto px-4 sm:px-6 lg:px-8 py-8 grid grid-cols-1 lg:grid-cols-3 gap-8">
      <div className="lg:col-span-2 space-y-6">
        <Skeleton className="w-full h-[576px] rounded-2xl" />
        <div className="space-y-4">
          <Skeleton className="h-8 w-3/4 rounded-lg" />
          <div className="flex justify-between items-center">
            <Skeleton className="h-4 w-1/4 rounded" />
            <Skeleton className="h-10 w-24 rounded-full" />
          </div>
        </div>
        <div className="mt-8 space-y-6">
          <Skeleton className="h-8 w-1/3 rounded-lg" />
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="flex items-center space-x-3">
                <Skeleton className="w-10 h-10 rounded-full flex-shrink-0" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-1/2 rounded" />
                  <Skeleton className="h-4 w-full rounded" />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className="lg:col-span-1 space-y-4">
        <Skeleton className="h-6 w-1/4 rounded-lg mb-4" />
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="flex space-x-3">
              <Skeleton className="w-40 h-24 rounded-xl flex-shrink-0" />
              <div className="flex-1 space-y-2 py-1">
                <Skeleton className="h-4 w-full rounded" />
                <Skeleton className="h-4 w-2/3 rounded" />
                <Skeleton className="h-3 w-1/2 rounded" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
};

export default VideoPlayerSkeleton;
