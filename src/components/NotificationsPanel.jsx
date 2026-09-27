"use client";

import React, { useState, useEffect, useRef } from "react";
import Link from "next/link";
import API from "@/lib/api";
import { useSession } from "next-auth/react";
import { FaBell } from "react-icons/fa";
import { toast } from "react-toastify";
import Image from "next/image";
import { getAvatarUrl } from "@/lib/identicon";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Card } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";

const NotificationsPanel = () => {
  const { data: session, status } = useSession();
  const isAuthenticated = status === "authenticated";
  const user = session?.user;

  const [notifications, setNotifications] = useState([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isOpen, setIsOpen] = useState(false);
  const panelRef = useRef(null);

  useEffect(() => {
    if (!isAuthenticated || !user) return;

    API.get("/notifications").then((res) => {
      setNotifications(res.data.notifications || []);
      setUnreadCount(res.data.unreadCount || 0);
    });

    let isMounted = true;
    let ws = null;
    let reconnectTimer = null;

    const connectWebSocket = () => {
      if (!isMounted) return;

      const serverUrl =
        process.env.NEXT_PUBLIC_NOTIFICATION_SERVER_URL ||
        "https://aurahub-go-notifier.onrender.com";
      const wsUrl = `${serverUrl.replace(/^http/, "ws")}/ws?userId=${user.id}`;

      try {
        ws = new WebSocket(wsUrl);

        ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            if (data && data._id) {
              setNotifications((prev) => [data, ...prev]);
              setUnreadCount((prev) => prev + 1);
              if (data?.sender?.username) {
                toast.info(`New notification from ${data.sender.username}!`);
              }
            }
          } catch {
            // Ignore non-JSON heartbeat
          }
        };

        ws.onerror = () => {};

        ws.onclose = () => {
          if (isMounted) {
            reconnectTimer = setTimeout(connectWebSocket, 5000);
          }
        };
      } catch {
        if (isMounted) {
          reconnectTimer = setTimeout(connectWebSocket, 5000);
        }
      }
    };

    connectWebSocket();

    return () => {
      isMounted = false;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (ws) {
        ws.onclose = null;
        ws.close();
      }
    };
  }, [isAuthenticated, user?.id]);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (panelRef.current && !panelRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleBellClick = async () => {
    setIsOpen(!isOpen);
    if (!isOpen && unreadCount > 0) {
      try {
        await API.post("/notifications");
        setUnreadCount(0);
        setNotifications((prev) => prev.map((n) => ({ ...n, isRead: true })));
      } catch (error) {
        console.error("Failed to mark notifications as read", error);
      }
    }
  };

  const handleClearAll = async () => {
    if (notifications.length === 0) return;
    try {
      await API.delete("/notifications");
      setNotifications([]);
      setUnreadCount(0);
      toast.success("Notifications cleared.");
    } catch (error) {
      toast.error("Failed to clear notifications.");
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="relative" ref={panelRef}>
      <Button
        variant="ghost"
        size="icon"
        onClick={handleBellClick}
        className="relative rounded-full text-muted-foreground hover:text-foreground"
      >
        <FaBell size={18} />
        {unreadCount > 0 && (
          <span className="absolute top-0 right-0 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground">
            {unreadCount}
          </span>
        )}
      </Button>

      {isOpen && (
        <Card className="absolute right-0 mt-2 w-80 sm:w-96 overflow-hidden shadow-xl border-border z-50 animate-in fade-in slide-in-from-top-2 duration-200">
          <div className="p-3 flex justify-between items-center border-b border-border bg-muted/50">
            <h3 className="font-bold text-sm text-foreground">Notifications</h3>
            {notifications.length > 0 && (
              <Button
                variant="link"
                size="sm"
                onClick={handleClearAll}
                className="h-auto p-0 text-xs text-foreground underline"
              >
                Clear All
              </Button>
            )}
          </div>
          <ScrollArea className="h-96">
            <ul className="flex flex-col">
              {notifications.length > 0 ? (
                notifications.map((notif) => (
                  <li key={notif._id}>
                    <Link
                      href={`/video/${notif.video?._id}`}
                      onClick={() => setIsOpen(false)}
                      className={`flex items-start gap-3 border-b border-border last:border-0 p-3 text-sm hover:bg-muted/50 transition-colors ${
                        !notif.isRead ? "bg-muted/80" : ""
                      }`}
                    >
                      <Avatar className="h-8 w-8 mt-1 flex-shrink-0">
                        <AvatarImage src={getAvatarUrl(notif.sender?.username, notif.sender?.avatar)} alt={notif.sender?.username} />
                        <AvatarFallback>{notif.sender?.username?.charAt(0) || "U"}</AvatarFallback>
                      </Avatar>
                      <div className="w-0 flex-grow text-foreground">
                        <p>
                          <strong className="font-semibold">
                            {notif.sender.username}
                          </strong>
                          {notif.type === "like" &&
                            ` liked your video: "${notif.video?.title}"`}
                          {notif.type === "comment" &&
                            ` commented on your video: "${notif.video?.title}"`}
                          {notif.type === "reply" && ` replied to your comment.`}
                        </p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {new Date(notif.createdAt).toLocaleString()}
                        </p>
                      </div>
                    </Link>
                  </li>
                ))
              ) : (
                <li className="p-4 text-center text-muted-foreground">
                  No new notifications.
                </li>
              )}
            </ul>
          </ScrollArea>
        </Card>
      )}
    </div>
  );
};

export default NotificationsPanel;
