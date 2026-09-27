"use client";

import React, { useState } from "react";
import Link from "next/link";
import useSWR from "swr";
import { fetcher } from "@/lib/fetcher";
import API from "@/lib/api";
import { toast } from "react-toastify";
import Comment from "@/components/Comment";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";

const VideoComments = ({ videoId, isAuthenticated }) => {
  const [newComment, setNewComment] = useState("");

  const { data: comments, mutate } = useSWR(
    `/videos/${videoId}/comments`,
    fetcher
  );

  const handleCommentSubmit = async (e) => {
    e.preventDefault();
    if (!newComment.trim()) return;
    try {
      const res = await API.post(`/videos/${videoId}/comments`, {
        text: newComment,
      });
      mutate([res.data, ...(comments || [])], false);
      setNewComment("");
      toast.success("Comment posted!");
    } catch (err) {
      toast.error("Failed to post comment. Please try again.");
    }
  };

  const handleCommentDeleted = (deletedComment) => {
    mutate(
      (comments || []).filter((c) => c._id !== deletedComment._id),
      false
    );
  };

  const handleCommentUpdated = (updatedComment) => {
    mutate(
      (comments || []).map((c) => (c._id === updatedComment._id ? updatedComment : c)),
      false
    );
  };

  if (!comments) {
    return <div className="mt-8 bg-card p-6 md:p-8 rounded-2xl shadow-xl border border-border animate-pulse h-64"></div>;
  }

  return (
    <div className="mt-8 bg-card p-6 md:p-8 rounded-2xl shadow-xl border border-border transition-colors duration-300">
      <h2 className="text-2xl font-bold text-foreground mb-6">
        {comments.length} Comments
      </h2>
      {isAuthenticated ? (
        <form onSubmit={handleCommentSubmit} className="mb-8">
          <Textarea
            value={newComment}
            onChange={(e) => setNewComment(e.target.value)}
            placeholder="Add a comment..."
            className="w-full resize-none"
            rows={3}
          />
          <Button
            type="submit"
            className="mt-3 rounded-xl px-6 font-semibold shadow-sm"
          >
            Post Comment
          </Button>
        </form>
      ) : (
        <p className="mb-8 text-muted-foreground font-medium">
          Please{" "}
          <Link href="/login" className="text-primary hover:underline">
            log in
          </Link>{" "}
          to post a comment.
        </p>
      )}
      <div className="space-y-4">
        {comments.map((comment) => (
          <Comment
            key={comment._id}
            comment={comment}
            videoId={videoId}
            onCommentDeleted={handleCommentDeleted}
            onCommentUpdated={handleCommentUpdated}
            onReplySubmitted={() => {}}
          />
        ))}
      </div>
    </div>
  );
};

export default VideoComments;
