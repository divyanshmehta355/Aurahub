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
      <div className="p-5 rounded-2xl bg-indigo-50/60 dark:bg-slate-900/60 border border-indigo-100 dark:border-slate-700/80 space-y-4">
        <div className="flex items-center space-x-2 text-indigo-900 dark:text-indigo-200 font-bold text-base sm:text-lg">
          <FaLayerGroup className="text-indigo-600 dark:text-indigo-400" />
          <span>Shared Playlist Upload Settings</span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          {/* Target Playlist */}
          <div>
            <label className="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">
              Add to Playlist
            </label>
            <select
              value={playlistOption}
              onChange={(e) => setPlaylistOption(e.target.value)}
              disabled={isRunning}
              className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-800 border border-gray-300 dark:border-slate-700 rounded-xl text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 transition-all cursor-pointer font-medium"
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
          <div>
            <label className="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">
              Category
            </label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              disabled={isRunning}
              className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-800 border border-gray-300 dark:border-slate-700 rounded-xl text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 transition-all cursor-pointer"
            >
              {CATEGORIES.map((cat) => (
                <option key={cat} value={cat}>
                  {cat}
                </option>
              ))}
            </select>
          </div>

          {/* Visibility */}
          <div>
            <label className="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">
              Visibility
            </label>
            <select
              value={visibility}
              onChange={(e) => setVisibility(e.target.value)}
              disabled={isRunning}
              className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-800 border border-gray-300 dark:border-slate-700 rounded-xl text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 transition-all cursor-pointer"
            >
              <option value="public">Public</option>
              <option value="unlisted">Unlisted</option>
              <option value="private">Private</option>
            </select>
          </div>
        </div>

        {/* Inline New Playlist Name Input */}
        {playlistOption === "create_new" && (
          <div className="p-3 bg-white dark:bg-slate-800 rounded-xl border border-indigo-200 dark:border-indigo-900/50 space-y-1 animate-in fade-in duration-200">
            <label className="block text-xs font-bold text-indigo-700 dark:text-indigo-400">
              New Playlist Name *
            </label>
            <div className="flex items-center space-x-2">
              <MdPlaylistAdd size={22} className="text-indigo-500 flex-shrink-0" />
              <input
                type="text"
                placeholder="e.g. Next.js 16 Masterclass, Season 1, etc."
                value={newPlaylistTitle}
                onChange={(e) => setNewPlaylistTitle(e.target.value)}
                disabled={isRunning}
                className="w-full px-3 py-1.5 text-sm bg-gray-50 dark:bg-slate-900 border border-gray-300 dark:border-slate-700 rounded-lg text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:outline-none"
              />
            </div>
          </div>
        )}

        {/* Shorts Toggle */}
        <div className="flex items-center space-x-2 pt-1">
          <input
            type="checkbox"
            id="playlistIsShort"
            checked={isShort}
            onChange={(e) => setIsShort(e.target.checked)}
            disabled={isRunning}
            className="w-4 h-4 text-indigo-600 rounded focus:ring-indigo-500 dark:bg-slate-900 border-gray-300 dark:border-slate-600 cursor-pointer"
          />
          <label
            htmlFor="playlistIsShort"
            className="text-xs font-medium text-gray-700 dark:text-gray-300 cursor-pointer"
          >
            Upload videos as Shorts (Vertical format)
          </label>
        </div>
      </div>

      {/* Action Header & Quick Paste Button */}
      <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
        <div>
          <h3 className="text-lg font-bold text-gray-900 dark:text-white">
            Playlist Videos Queue ({items.length})
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Enter each video's Title and Remote MP4/stream URL. They will be downloaded and added to your playlist one by one.
          </p>
        </div>

        <button
          type="button"
          onClick={() => setShowPasteModal(true)}
          disabled={isRunning}
          className="flex items-center space-x-1.5 text-xs font-semibold px-3 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-gray-700 dark:text-gray-300 rounded-xl transition-all disabled:opacity-50"
        >
          <FaClipboardList className="text-indigo-500" />
          <span>Paste Batch URLs</span>
        </button>
      </div>

      {/* Video Row Items List */}
      <div className="space-y-3 max-h-[55vh] overflow-y-auto pr-1 custom-scrollbar">
        {items.map((item, index) => {
          const isCurrent = currentProcessingIndex === index;
          return (
            <div
              key={item.id}
              className={`p-4 rounded-xl border transition-all duration-300 ${
                isCurrent
                  ? "bg-indigo-50/70 dark:bg-slate-800/90 border-indigo-400 ring-2 ring-indigo-500/20 shadow-md"
                  : item.status === "completed"
                  ? "bg-emerald-50/50 dark:bg-emerald-950/20 border-emerald-300 dark:border-emerald-800/60"
                  : item.status === "error"
                  ? "bg-rose-50/50 dark:bg-rose-950/20 border-rose-300 dark:border-rose-800/60"
                  : "bg-white dark:bg-slate-800/80 border-gray-200 dark:border-slate-700 shadow-sm"
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center space-x-2">
                  <span className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold bg-gray-100 dark:bg-slate-700 text-gray-700 dark:text-gray-300">
                    {index + 1}
                  </span>
                  <span className="text-xs font-semibold text-gray-500 dark:text-gray-400">
                    Video #{index + 1}
                  </span>
                </div>

                <div className="flex items-center space-x-2">
                  {/* Status Indicator */}
                  {item.status === "idle" && (
                    <span className="text-[11px] font-medium text-gray-500 dark:text-gray-400 px-2 py-0.5 rounded-full bg-gray-100 dark:bg-slate-700">
                      Ready
                    </span>
                  )}
                  {item.status === "queuing" && (
                    <span className="flex items-center space-x-1 text-[11px] font-medium text-indigo-600 dark:text-indigo-400 px-2 py-0.5 rounded-full bg-indigo-100 dark:bg-indigo-900/40">
                      <FaSpinner className="animate-spin text-[10px]" />
                      <span>Queuing...</span>
                    </span>
                  )}
                  {item.status === "downloading" && (
                    <span className="flex items-center space-x-1 text-[11px] font-semibold text-cyan-700 dark:text-cyan-300 px-2 py-0.5 rounded-full bg-cyan-100 dark:bg-cyan-900/40">
                      <FaSpinner className="animate-spin text-[10px]" />
                      <span>Downloading {item.progress}%</span>
                    </span>
                  )}
                  {item.status === "publishing" && (
                    <span className="flex items-center space-x-1 text-[11px] font-semibold text-purple-700 dark:text-purple-300 px-2 py-0.5 rounded-full bg-purple-100 dark:bg-purple-900/40">
                      <FaSpinner className="animate-spin text-[10px]" />
                      <span>Publishing...</span>
                    </span>
                  )}
                  {item.status === "completed" && (
                    <span className="flex items-center space-x-1 text-[11px] font-bold text-emerald-700 dark:text-emerald-300 px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-900/40">
                      <FaCheckCircle className="text-[10px]" />
                      <span>Completed</span>
                    </span>
                  )}
                  {item.status === "error" && (
                    <span className="flex items-center space-x-1 text-[11px] font-bold text-rose-700 dark:text-rose-300 px-2 py-0.5 rounded-full bg-rose-100 dark:bg-rose-900/40" title={item.error}>
                      <FaExclamationCircle className="text-[10px]" />
                      <span className="truncate max-w-[120px]">Failed</span>
                    </span>
                  )}

                  {/* Delete Row Button */}
                  {!isRunning && items.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeRow(item.id)}
                      className="p-1.5 text-gray-400 hover:text-rose-500 rounded-lg transition-colors"
                      title="Remove video row"
                    >
                      <FaTrashAlt size={13} />
                    </button>
                  )}
                </div>
              </div>

              {/* Title & URL Inputs */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <input
                    type="text"
                    placeholder="Video Title *"
                    value={item.title}
                    onChange={(e) => updateItem(item.id, { title: e.target.value })}
                    disabled={isRunning}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-900 border border-gray-300 dark:border-slate-700 rounded-xl text-gray-900 dark:text-white placeholder-gray-400 focus:ring-2 focus:ring-indigo-500 focus:outline-none transition-all"
                  />
                </div>
                <div>
                  <input
                    type="url"
                    placeholder="Remote URL (http://...mp4) *"
                    value={item.videoUrl}
                    onChange={(e) => updateItem(item.id, { videoUrl: e.target.value })}
                    onBlur={(e) => handleUrlBlur(item.id, e.target.value)}
                    disabled={isRunning}
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-900 border border-gray-300 dark:border-slate-700 rounded-xl text-gray-900 dark:text-white placeholder-gray-400 focus:ring-2 focus:ring-indigo-500 focus:outline-none transition-all font-mono text-xs sm:text-sm"
                  />
                </div>
              </div>

              {/* Active Item Progress Bar */}
              {isCurrent && item.status === "downloading" && (
                <div className="mt-3 space-y-1">
                  <div className="w-full bg-gray-200 dark:bg-slate-700 rounded-full h-1.5 overflow-hidden">
                    <div
                      className="bg-gradient-to-r from-indigo-500 to-cyan-400 h-1.5 rounded-full transition-all duration-300"
                      style={{ width: `${item.progress}%` }}
                    />
                  </div>
                </div>
              )}

              {/* Error Message if any */}
              {item.error && (
                <p className="mt-2 text-xs text-rose-500 dark:text-rose-400 flex items-center space-x-1">
                  <FaExclamationCircle size={11} className="flex-shrink-0" />
                  <span>{item.error}</span>
                </p>
              )}
            </div>
          );
        })}
      </div>

      {/* Add Row Button */}
      {!isRunning && (
        <button
          type="button"
          onClick={addRow}
          className="w-full py-2.5 border-2 border-dashed border-gray-300 dark:border-slate-700 hover:border-indigo-500 dark:hover:border-indigo-500 text-gray-600 dark:text-gray-300 hover:text-indigo-600 dark:hover:text-indigo-400 font-bold rounded-xl flex items-center justify-center space-x-2 text-sm transition-all duration-200"
        >
          <FaPlus size={12} />
          <span>Add Another Video</span>
        </button>
      )}

      {/* Overall Progress Bar during Execution */}
      {isRunning && (
        <div className="p-4 rounded-xl bg-indigo-50 dark:bg-slate-900/90 border border-indigo-100 dark:border-slate-700 space-y-2 animate-in fade-in duration-300">
          <div className="flex justify-between text-xs font-bold text-gray-700 dark:text-gray-300">
            <span>Overall Progress</span>
            <span>
              {completedCount} / {validCount} Completed ({overallPercent}%)
            </span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-slate-700 rounded-full h-2.5 overflow-hidden">
            <div
              className="bg-indigo-600 dark:bg-indigo-500 h-2.5 rounded-full transition-all duration-500 relative"
              style={{ width: `${overallPercent}%` }}
            >
              <div className="absolute inset-0 bg-white/20 animate-pulse" />
            </div>
          </div>
          <p className="text-xs text-gray-500 dark:text-gray-400 text-center animate-pulse">
            Processing video {currentProcessingIndex + 1} of {validCount}... please keep this tab open.
          </p>
        </div>
      )}

      {/* Completion Banner with Playlist Link */}
      {!isRunning && createdPlaylist && completedCount > 0 && (
        <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800 text-emerald-900 dark:text-emerald-200 flex flex-col sm:flex-row items-center justify-between gap-3 animate-in fade-in duration-300">
          <div className="flex items-center space-x-2">
            <MdCheck size={24} className="text-emerald-500" />
            <span className="text-sm font-semibold">
              Playlist <strong>"{createdPlaylist.title}"</strong> is ready with your videos!
            </span>
          </div>
          <Link
            href={`/playlist/${createdPlaylist._id}`}
            className="text-xs font-bold px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors shadow-sm"
          >
            View Playlist →
          </Link>
        </div>
      )}

      {/* Bottom Main Action Button */}
      <div className="flex gap-3 pt-2">
        {isRunning ? (
          <button
            type="button"
            onClick={handleStop}
            className="w-full py-3.5 px-4 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl transition-all shadow-md flex items-center justify-center space-x-2"
          >
            <FaStop />
            <span>Stop Uploads</span>
          </button>
        ) : (
          <button
            type="button"
            onClick={handleStartPlaylistUpload}
            disabled={validCount === 0}
            className="w-full py-3.5 px-4 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 text-white font-bold rounded-xl transition-all shadow-md hover:shadow-lg transform hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center space-x-2 text-base"
          >
            <FaPlay size={14} />
            <span>
              Start Playlist Upload {validCount > 0 ? `(${validCount} Videos)` : ""}
            </span>
          </button>
        )}
      </div>

      {/* Quick Paste Modal */}
      {showPasteModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in duration-200">
          <div className="bg-white dark:bg-slate-900 border border-gray-200 dark:border-slate-800 rounded-2xl max-w-lg w-full p-6 shadow-2xl space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-bold text-gray-900 dark:text-white flex items-center space-x-2">
                <FaClipboardList className="text-indigo-600" />
                <span>Paste Multiple Video URLs</span>
              </h3>
              <button
                type="button"
                onClick={() => setShowPasteModal(false)}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 text-lg"
              >
                ✕
              </button>
            </div>

            <p className="text-xs text-gray-500 dark:text-gray-400">
              Paste URLs one per line. You can paste just URLs, or formatted as <code>Title - URL</code> or <code>Title, URL</code>:
            </p>

            <textarea
              rows={8}
              value={rawPastedText}
              onChange={(e) => setRawPastedText(e.target.value)}
              placeholder={`Episode 1 - https://example.com/ep1.mp4\nEpisode 2 - https://example.com/ep2.mp4\nhttps://example.com/ep3.mp4`}
              className="w-full p-3 text-xs sm:text-sm font-mono bg-gray-50 dark:bg-slate-800 border border-gray-300 dark:border-slate-700 rounded-xl text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:outline-none"
            />

            <div className="flex justify-end space-x-3">
              <button
                type="button"
                onClick={() => setShowPasteModal(false)}
                className="px-4 py-2 text-xs font-semibold text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleBatchPaste}
                disabled={!rawPastedText.trim()}
                className="px-4 py-2 text-xs font-bold bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition-colors shadow-md disabled:opacity-50"
              >
                Populate Videos
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default PlaylistUploadForm;
