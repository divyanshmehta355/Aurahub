import React from "react";
import { Skeleton } from "@/components/ui/skeleton";

const VideoCardSkeleton = () => {
  return (
    <div className="flex flex-col gap-2">
      <Skeleton className="w-full h-48 rounded-lg" />
      <div className="flex gap-3 mt-1">
        <Skeleton className="w-10 h-10 rounded-full flex-shrink-0" />
        <div className="flex flex-col gap-2 w-full">
          <Skeleton className="w-full h-5" />
          <Skeleton className="w-2/3 h-4" />
        </div>
      </div>
    </div>
  );
};

export default VideoCardSkeleton;
