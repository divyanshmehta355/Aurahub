"use client";

import React, { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { usePathname, useRouter } from "next/navigation";
import { useAppStore } from "@/store/useAppStore";
import { FaTimes, FaExpandAlt } from "react-icons/fa";
import { Button } from "@/components/ui/button";

const GlobalVideoPlayer = () => {
  const activeVideo = useAppStore((state) => state.activeVideo);
  const closeVideo = useAppStore((state) => state.closeVideo);
  const pathname = usePathname();
  const router = useRouter();
  
  const [slotElement, setSlotElement] = useState(null);

  const isMainVideoPage = activeVideo && pathname === `/video/${activeVideo.id}`;

  useEffect(() => {
    let intervalId;
    if (isMainVideoPage) {
      let attempts = 0;
      const findSlot = () => {
        attempts++;
        const el = document.getElementById("video-player-slot");
        if (el) {
          setSlotElement(el);
          clearInterval(intervalId);
        } else if (attempts >= 30) {
          clearInterval(intervalId);
        }
      };
      
      findSlot();
      intervalId = setInterval(findSlot, 50);
    } else {
      setSlotElement(null);
    }

    return () => clearInterval(intervalId);
  }, [isMainVideoPage, pathname]);

  if (!activeVideo) return null;

  const iframeContent = (
    <iframe
      src={`https://streamtape.com/e/${activeVideo.fileId}`}
      title={activeVideo.title}
      frameBorder="0"
      allowFullScreen
      className="w-full h-full border-0"
    ></iframe>
  );

  if (isMainVideoPage && slotElement) {
    return createPortal(iframeContent, slotElement);
  }

  if (isMainVideoPage && !slotElement) {
    return null;
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 w-80 sm:w-96 shadow-2xl rounded-xl overflow-hidden bg-foreground ring-1 ring-white/10 group animate-in slide-in-from-bottom-8 fade-in duration-300">
      <div className="relative w-full aspect-video bg-foreground">
        {iframeContent}
        
        <div className="absolute top-0 left-0 w-full h-full opacity-0 group-hover:opacity-100 transition-opacity bg-foreground/40 pointer-events-none flex flex-col justify-between p-2">
          <div className="flex justify-end gap-2 pointer-events-auto">
             <Button
                variant="ghost"
                size="icon"
                onClick={() => router.push(`/video/${activeVideo.id}`)}
                className="h-8 w-8 bg-foreground/60 hover:bg-foreground/80 rounded-full text-primary-foreground backdrop-blur-md transition-all hover:scale-105"
                title="Expand"
             >
                <FaExpandAlt size={14} />
             </Button>
             <Button
                variant="ghost"
                size="icon"
                onClick={closeVideo}
                className="h-8 w-8 bg-foreground/60 hover:bg-foreground/80 rounded-full text-primary-foreground backdrop-blur-md transition-all hover:scale-105"
                title="Close Miniplayer"
             >
                <FaTimes size={14} />
             </Button>
          </div>
          <div className="pointer-events-auto bg-gradient-to-t from-black/80 to-transparent p-2 -mx-2 -mb-2 mt-auto">
             <p className="text-primary-foreground text-sm font-medium truncate drop-shadow-md">
               {activeVideo.title}
             </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default GlobalVideoPlayer;
