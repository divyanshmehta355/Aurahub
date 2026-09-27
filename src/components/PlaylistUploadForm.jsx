"use client";

import React, { useState, useRef, useEffect } from "react";
import Link from "next/link";
import API from "@/lib/api";
import CATEGORIES from "@/constants/categories";
import { toast } from "react-toastify";
import {
  FaPlus,
  FaTrashAlt,
  FaCheckCircle,
  FaExclamationCircle,
  FaSpinner,
  FaLayerGroup,
  FaPlay,
  FaStop,
  FaClipboardList,
} from "react-icons/fa";
import { MdPlaylistAdd, MdCheck } from "react-icons/md";

import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Progress } from "@/components/ui/progress";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

// Utility to convert file URL to a clean default title
function extractTitleFromUrl(url) {
  try {
    const cleanUrl = url.split("?")[0];
    const filename = cleanUrl.substring(cleanUrl.lastIndexOf("/") + 1);
    const withoutExt = filename.replace(/\.[a-zA-Z0-9]+$/, "");
    return decodeURIComponent(withoutExt)
      .replace(/[-_]+/g, " ")
      .replace(/\s+/g, " ")
      .trim();
  } catch {
    return "";
  }
}

const PlaylistUploadForm = ({ playlists = [], onPlaylistCreated }) => {
  // Shared Configuration
  const [playlistOption, setPlaylistOption] = useState("create_new"); // "create_new", "existing", "none"
  const [selectedPlaylistId, setSelectedPlaylistId] = useState("");
  const [newPlaylistTitle, setNewPlaylistTitle] = useState("");
  const [category, setCategory] = useState("Other");
  const [visibility, setVisibility] = useState("public");
  const [isShort, setIsShort] = useState(false);

  // Dynamic Video Rows
  const [items, setItems] = useState([
    { id: "1", title: "", videoUrl: "", status: "idle", progress: 0, error: "" },
    { id: "2", title: "", videoUrl: "", status: "idle", progress: 0, error: "" },
  ]);

  // Playlist Upload Queue Execution State
  const [isRunning, setIsRunning] = useState(false);
  const [currentProcessingIndex, setCurrentProcessingIndex] = useState(-1);
  const [completedCount, setCompletedCount] = useState(0);
  const [createdPlaylist, setCreatedPlaylist] = useState(null);
  const [showPasteModal, setShowPasteModal] = useState(false);
  const [rawPastedText, setRawPastedText] = useState("");

  const abortRef = useRef(false);
  const pollingIntervalRef = useRef(null);

  useEffect(() => {
    return () => {
      abortRef.current = true;
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, []);

  // Helpers to update individual rows
  const updateItem = (id, updates) => {
    setItems((prev) =>
      prev.map((item) => (item.id === id ? { ...item, ...updates } : item))
    );
  };

  const handleUrlBlur = (id, url) => {
    const item = items.find((i) => i.id === id);
    if (item && !item.title.trim() && url.trim()) {
      const autoTitle = extractTitleFromUrl(url);
      if (autoTitle) {
        updateItem(id, { title: autoTitle });
      }
    }
  };

  const addRow = () => {
    const newId = Date.now().toString() + Math.random().toString(36).substring(5);
    setItems((prev) => [
      ...prev,
      { id: newId, title: "", videoUrl: "", status: "idle", progress: 0, error: "" },
    ]);
  };

  const removeRow = (id) => {
    if (items.length <= 1) {
      toast.warn("At least one video row is required.");
      return;
    }
    setItems((prev) => prev.filter((item) => item.id !== id));
  };

  const handleBatchPaste = () => {
    if (!rawPastedText.trim()) return;

    const lines = rawPastedText
      .split("\n")
      .map((l) => l.trim())
      .filter(Boolean);

    const newRows = [];

    lines.forEach((line) => {
      let parsedTitle = "";
      let parsedUrl = "";

      const splitDash = line.split(/\s+-\s+/);
      const splitPipe = line.split(/\s+\|\s+/);
      const splitComma = line.split(/\s*,\s*(?=https?:\/\/)/i);

      if (splitDash.length === 2 && splitDash[1].startsWith("http")) {
        parsedTitle = splitDash[0].trim();
        parsedUrl = splitDash[1].trim();
      } else if (splitPipe.length === 2 && splitPipe[1].startsWith("http")) {
        parsedTitle = splitPipe[0].trim();
        parsedUrl = splitPipe[1].trim();
      } else if (splitComma.length === 2 && splitComma[1].startsWith("http")) {
        parsedTitle = splitComma[0].trim();
        parsedUrl = splitComma[1].trim();
      } else if (line.startsWith("http")) {
        parsedUrl = line;
        parsedTitle = extractTitleFromUrl(line);
      } else {
        parsedTitle = line;
      }

      if (parsedUrl || parsedTitle) {
        newRows.push({
          id: Date.now().toString() + Math.random().toString(36).substring(5),
          title: parsedTitle || "Untitled Video",
          videoUrl: parsedUrl || "",
          status: "idle",
          progress: 0,
          error: "",
        });
      }
    });

    if (newRows.length > 0) {
      const isEmpty = items.length === 2 && !items[0].videoUrl && !items[1].videoUrl;
      setItems((prev) => (isEmpty ? newRows : [...prev, ...newRows]));
      toast.success(`Imported ${newRows.length} videos from text!`);
    }

    setRawPastedText("");
    setShowPasteModal(false);
  };

  // Poll Remote Upload until finished
  const pollRemoteUpload = (remoteId, onProgress) => {
    return new Promise((resolve, reject) => {
      let failedCount = 0;
      const maxFailed = 8;

      pollingIntervalRef.current = setInterval(async () => {
        if (abortRef.current) {
          clearInterval(pollingIntervalRef.current);
          return reject(new Error("Upload stopped by user."));
        }

        try {
          const statusRes = await API.get("/videos/remote-upload/status", {
            params: { id: remoteId },
          });

          failedCount = 0;
          const statusData = statusRes.data?.[remoteId];

          if (statusData && statusData.status === "finished") {
            clearInterval(pollingIntervalRef.current);
            onProgress(100);
            return resolve(statusData.linkid);
          } else if (statusData && statusData.status === "error") {
            clearInterval(pollingIntervalRef.current);
            return reject(
              new Error(statusData.error_message || "Remote download error")
            );
          } else if (statusData) {
            const loaded = statusData.bytes_loaded || 0;
            const total = statusData.bytes_total || 0;
            const pct = total > 0 ? Math.floor((loaded * 100) / total) : 10;
            onProgress(pct);
          }
        } catch (err) {
          failedCount++;
          if (failedCount >= maxFailed) {
            clearInterval(pollingIntervalRef.current);
            return reject(
              new Error("Lost connection to remote downloader status.")
            );
          }
        }
      }, 4000);
    });
  };

  // Main Playlist Sequential Queue Runner
  const handleStartPlaylistUpload = async () => {
    // 1. Validation
    const validItems = items.filter(
      (item) => item.title.trim() && item.videoUrl.trim()
    );

    if (validItems.length === 0) {
      toast.error("Please enter a Title and Remote URL for at least one video.");
      return;
    }

    if (playlistOption === "create_new" && !newPlaylistTitle.trim()) {
      toast.error("Please enter a name for the new playlist.");
      return;
    }

    setIsRunning(true);
    abortRef.current = false;
    let targetPlaylistId = null;

    // 2. Create playlist if requested
    if (playlistOption === "create_new" && newPlaylistTitle.trim()) {
      try {
        toast.info(`Creating playlist "${newPlaylistTitle.trim()}"...`);
        const plRes = await API.post("/playlists", {
          title: newPlaylistTitle.trim(),
          isPublic: visibility === "public",
        });
        targetPlaylistId = plRes.data?._id;
        setCreatedPlaylist(plRes.data);
        if (onPlaylistCreated) {
          onPlaylistCreated(plRes.data);
        }
        toast.success(`Playlist created!`);
      } catch (plErr) {
        toast.error("Failed to create playlist. Continuing upload without playlist.");
      }
    } else if (playlistOption === "existing" && selectedPlaylistId) {
      targetPlaylistId = selectedPlaylistId;
    }

    let successfulUploads = 0;

    // 3. Process videos sequentially one by one
    for (let i = 0; i < items.length; i++) {
      if (abortRef.current) break;

      const item = items[i];

      // Skip blank rows or already completed items
      if (!item.title.trim() || !item.videoUrl.trim()) continue;
      if (item.status === "completed") {
        successfulUploads++;
        continue;
      }

      setCurrentProcessingIndex(i);
      updateItem(item.id, { status: "queuing", progress: 0, error: "" });

      try {
        // Step A: Trigger remote upload start
        const startRes = await API.post("/videos/remote-upload/start", {
          videoUrl: item.videoUrl.trim(),
        });
        const remoteId = startRes.data?.id;

        if (!remoteId) {
          throw new Error("Did not receive a remote upload ID from server.");
        }

        // Step B: Poll until finished
        updateItem(item.id, { status: "downloading", progress: 5 });

        const videoId = await pollRemoteUpload(remoteId, (percent) => {
          updateItem(item.id, { progress: percent });
        });

        if (abortRef.current) break;

        // Step C: Create Video Record in MongoDB
        updateItem(item.id, { status: "publishing", progress: 99 });

        const finalFormData = new FormData();
        finalFormData.append("title", item.title.trim());
        finalFormData.append("description", ""); // optional
        finalFormData.append("videoId", videoId);
        finalFormData.append("category", category);
        finalFormData.append("visibility", visibility);
        finalFormData.append("isShort", Boolean(isShort));
        if (targetPlaylistId) {
          finalFormData.append("playlistId", targetPlaylistId);
        }

        await API.post("/videos/create-record", finalFormData);

        updateItem(item.id, { status: "completed", progress: 100 });
        successfulUploads++;
        setCompletedCount(successfulUploads);
        toast.success(`"${item.title.trim()}" published!`);
      } catch (err) {
        console.error(`Error uploading "${item.title}":`, err);
        updateItem(item.id, {
          status: "error",
          error: err.message || "Failed to process remote upload",
        });
        toast.error(`Error with "${item.title}": ${err.message || "Upload failed"}`);
      }
    }

    setIsRunning(false);
    setCurrentProcessingIndex(-1);

    if (abortRef.current) {
      toast.warn("Playlist upload stopped.");
    } else {
      toast.success(`🎉 Playlist upload completed! (${successfulUploads} videos)`);
    }
  };

  const handleStop = () => {
    abortRef.current = true;
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
    }
    setIsRunning(false);
    setCurrentProcessingIndex(-1);
    toast.info("Stopping remaining uploads...");
  };

  const validCount = items.filter((i) => i.title.trim() && i.videoUrl.trim()).length;
  const overallPercent = validCount > 0 ? Math.floor((completedCount * 100) / validCount) : 0;

  return (
    <div className="space-y-6 animate-in fade-in duration-300">
      {/* Configuration Header Card */}
      <Card className="p-5 bg-muted/30 space-y-4">
        <div className="flex items-center space-x-2 font-bold text-base sm:text-lg">
          <FaLayerGroup className="text-muted-foreground" />
          <span>Shared Playlist Upload Settings</span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          {/* Target Playlist */}
          <div className="space-y-1.5">
            <Label>Add to Playlist</Label>
            <select
              value={playlistOption}
              onChange={(e) => {
                setPlaylistOption(e.target.value);
                if (e.target.value !== "create_new" && e.target.value !== "none") {
                    setSelectedPlaylistId(e.target.value);
                }
              }}
              disabled={isRunning}
              className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="create_new">+ Create New Playlist (Recommended)</option>
              {playlists.length > 0 && (
                <optgroup label="Your Existing Playlists">
                  {playlists.map((p) => (
                    <option key={p._id} value={p._id}>
                      {p.title}
                    </option>
                  ))}
                </optgroup>
              )}
              <option value="none">None (Individual Videos)</option>
            </select>
          </div>

          {/* Category */}
          <div className="space-y-1.5">
            <Label>Category</Label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              disabled={isRunning}
              className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {CATEGORIES.map((cat) => (
                <option key={cat} value={cat}>
                  {cat}
                </option>
              ))}
            </select>
          </div>

          {/* Visibility */}
          <div className="space-y-1.5">
            <Label>Visibility</Label>
            <select
              value={visibility}
              onChange={(e) => setVisibility(e.target.value)}
              disabled={isRunning}
              className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="public">Public</option>
              <option value="unlisted">Unlisted</option>
              <option value="private">Private</option>
            </select>
          </div>
        </div>

        {/* Inline New Playlist Name Input */}
        {playlistOption === "create_new" && (
          <div className="p-3 bg-background rounded-xl border border-border space-y-1.5 animate-in fade-in duration-200">
            <Label>
              New Playlist Name *
            </Label>
            <div className="flex items-center space-x-2">
              <MdPlaylistAdd size={22} className="text-muted-foreground flex-shrink-0" />
              <Input
                type="text"
                placeholder="e.g. Next.js 16 Masterclass, Season 1, etc."
                value={newPlaylistTitle}
                onChange={(e) => setNewPlaylistTitle(e.target.value)}
                disabled={isRunning}
                className="w-full bg-muted/50 border-input"
              />
            </div>
          </div>
        )}

        {/* Shorts Toggle */}
        <div className="flex items-center space-x-2 pt-1">
          <Checkbox
            id="playlistIsShort"
            checked={isShort}
            onCheckedChange={(checked) => setIsShort(checked)}
            disabled={isRunning}
          />
          <Label htmlFor="playlistIsShort" className="cursor-pointer">
            Upload videos as Shorts (Vertical format)
          </Label>
        </div>
      </Card>

      {/* Action Header & Quick Paste Button */}
      <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
        <div>
          <h3 className="text-lg font-bold text-foreground">
            Playlist Videos Queue ({items.length})
          </h3>
          <p className="text-xs text-muted-foreground">
            Enter each video's Title and Remote MP4/stream URL. They will be downloaded and added to your playlist one by one.
          </p>
        </div>

        <Button
          variant="secondary"
          size="sm"
          onClick={() => setShowPasteModal(true)}
          disabled={isRunning}
          className="flex items-center space-x-1.5 font-semibold"
        >
          <FaClipboardList />
          <span>Paste Batch URLs</span>
        </Button>
      </div>

      {/* Video Row Items List */}
      <div className="space-y-3 max-h-[55vh] overflow-y-auto pr-1 custom-scrollbar">
        {items.map((item, index) => {
          const isCurrent = currentProcessingIndex === index;
          return (
            <Card
              key={item.id}
              className={`p-4 transition-all duration-300 ${
                isCurrent
                  ? "bg-muted/80 ring-2 ring-primary/20"
                  : item.status === "completed"
                  ? "bg-primary/5 border-primary/20"
                  : item.status === "error"
                  ? "bg-destructive/10 border-destructive/20"
                  : "bg-card"
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center space-x-2">
                  <span className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold bg-muted text-muted-foreground">
                    {index + 1}
                  </span>
                  <span className="text-xs font-semibold text-muted-foreground">
                    Video #{index + 1}
                  </span>
                </div>

                <div className="flex items-center space-x-2">
                  {/* Status Indicator */}
                  {item.status === "idle" && (
                    <span className="text-[11px] font-medium text-muted-foreground px-2 py-0.5 rounded-full bg-muted">
                      Ready
                    </span>
                  )}
                  {item.status === "queuing" && (
                    <span className="flex items-center space-x-1 text-[11px] font-medium px-2 py-0.5 rounded-full bg-muted text-muted-foreground">
                      <FaSpinner className="animate-spin text-[10px]" />
                      <span>Queuing...</span>
                    </span>
                  )}
                  {item.status === "downloading" && (
                    <span className="flex items-center space-x-1 text-[11px] font-semibold px-2 py-0.5 rounded-full bg-muted text-foreground">
                      <FaSpinner className="animate-spin text-[10px]" />
                      <span>Downloading {item.progress}%</span>
                    </span>
                  )}
                  {item.status === "publishing" && (
                    <span className="flex items-center space-x-1 text-[11px] font-semibold px-2 py-0.5 rounded-full bg-muted text-foreground">
                      <FaSpinner className="animate-spin text-[10px]" />
                      <span>Publishing...</span>
                    </span>
                  )}
                  {item.status === "completed" && (
                    <span className="flex items-center space-x-1 text-[11px] font-bold px-2 py-0.5 rounded-full bg-primary text-primary-foreground">
                      <FaCheckCircle className="text-[10px]" />
                      <span>Completed</span>
                    </span>
                  )}
                  {item.status === "error" && (
                    <span className="flex items-center space-x-1 text-[11px] font-bold text-destructive px-2 py-0.5 rounded-full bg-destructive/10" title={item.error}>
                      <FaExclamationCircle className="text-[10px]" />
                      <span className="truncate max-w-[120px]">Failed</span>
                    </span>
                  )}

                  {/* Delete Row Button */}
                  {!isRunning && items.length > 1 && (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => removeRow(item.id)}
                      className="h-7 w-7 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                      title="Remove video row"
                    >
                      <FaTrashAlt size={13} />
                    </Button>
                  )}
                </div>
              </div>

              {/* Title & URL Inputs */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <Input
                  type="text"
                  placeholder="Video Title *"
                  value={item.title}
                  onChange={(e) => updateItem(item.id, { title: e.target.value })}
                  disabled={isRunning}
                />
                <Input
                  type="url"
                  placeholder="Remote URL (http://...mp4) *"
                  value={item.videoUrl}
                  onChange={(e) => updateItem(item.id, { videoUrl: e.target.value })}
                  onBlur={(e) => handleUrlBlur(item.id, e.target.value)}
                  disabled={isRunning}
                  className="font-mono text-xs sm:text-sm"
                />
              </div>

              {/* Active Item Progress Bar */}
              {isCurrent && item.status === "downloading" && (
                <div className="mt-3 space-y-1">
                  <Progress value={item.progress} className="h-1.5" />
                </div>
              )}

              {/* Error Message if any */}
              {item.error && (
                <p className="mt-2 text-xs text-destructive flex items-center space-x-1">
                  <FaExclamationCircle size={11} className="flex-shrink-0" />
                  <span>{item.error}</span>
                </p>
              )}
            </Card>
          );
        })}
      </div>

      {/* Add Row Button */}
      {!isRunning && (
        <Button
          variant="outline"
          onClick={addRow}
          className="w-full h-12 border-dashed border-2 font-bold"
        >
          <FaPlus size={12} className="mr-2" />
          Add Another Video
        </Button>
      )}

      {/* Overall Progress Bar during Execution */}
      {isRunning && (
        <Card className="p-4 bg-muted/50 space-y-2 animate-in fade-in duration-300">
          <div className="flex justify-between text-xs font-bold text-foreground">
            <span>Overall Progress</span>
            <span>
              {completedCount} / {validCount} Completed ({overallPercent}%)
            </span>
          </div>
          <Progress value={overallPercent} className="h-2.5" />
          <p className="text-xs text-muted-foreground text-center animate-pulse">
            Processing video {currentProcessingIndex + 1} of {validCount}... please keep this tab open.
          </p>
        </Card>
      )}

      {/* Completion Banner with Playlist Link */}
      {!isRunning && createdPlaylist && completedCount > 0 && (
        <Card className="p-4 bg-primary/10 border-primary text-foreground flex flex-col sm:flex-row items-center justify-between gap-3 animate-in fade-in duration-300">
          <div className="flex items-center space-x-2">
            <MdCheck size={24} className="text-primary" />
            <span className="text-sm font-semibold">
              Playlist <strong>"{createdPlaylist.title}"</strong> is ready with your videos!
            </span>
          </div>
          <Button asChild>
            <Link href={`/playlist/${createdPlaylist._id}`}>
              View Playlist →
            </Link>
          </Button>
        </Card>
      )}

      {/* Bottom Main Action Button */}
      <div className="flex gap-3 pt-2">
        {isRunning ? (
          <Button
            variant="destructive"
            onClick={handleStop}
            className="w-full h-14 text-base font-bold"
          >
            <FaStop className="mr-2" />
            Stop Uploads
          </Button>
        ) : (
          <Button
            onClick={handleStartPlaylistUpload}
            disabled={validCount === 0}
            className="w-full h-14 text-base font-bold"
          >
            <FaPlay size={14} className="mr-2" />
            Start Playlist Upload {validCount > 0 ? `(${validCount} Videos)` : ""}
          </Button>
        )}
      </div>

      {/* Quick Paste Modal */}
      <Dialog open={showPasteModal} onOpenChange={setShowPasteModal}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center space-x-2">
              <FaClipboardList className="text-muted-foreground" />
              <span>Paste Multiple Video URLs</span>
            </DialogTitle>
            <DialogDescription>
              Paste URLs one per line. You can paste just URLs, or formatted as <code>Title - URL</code> or <code>Title, URL</code>:
            </DialogDescription>
          </DialogHeader>

          <Textarea
            rows={8}
            value={rawPastedText}
            onChange={(e) => setRawPastedText(e.target.value)}
            placeholder={`Episode 1 - https://example.com/ep1.mp4\nEpisode 2 - https://example.com/ep2.mp4\nhttps://example.com/ep3.mp4`}
            className="font-mono text-xs sm:text-sm resize-none"
          />

          <DialogFooter>
            <Button variant="ghost" onClick={() => setShowPasteModal(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleBatchPaste}
              disabled={!rawPastedText.trim()}
            >
              Populate Videos
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default PlaylistUploadForm;
