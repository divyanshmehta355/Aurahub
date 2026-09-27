"use client";

import React, { useState } from 'react';
import Link from 'next/link';
import API from '@/lib/api';
import { useSession } from 'next-auth/react';
import { toast } from 'react-toastify';
import { getAvatarUrl } from '@/lib/identicon';

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";

const Comment = ({ comment, videoId, onCommentDeleted, onCommentUpdated, onReplySubmitted }) => {
    const [showReplyForm, setShowReplyForm] = useState(false);
    const [isEditing, setIsEditing] = useState(false);
    const [editText, setEditText] = useState(comment.text);
    const [replyText, setReplyText] = useState('');
    const [replies, setReplies] = useState([]);
    const [loadingReplies, setLoadingReplies] = useState(false);
    
    const { data: session, status } = useSession();
    const isAuthenticated = status === "authenticated";
    const currentUser = session?.user;
    const isOwner = currentUser?.id === comment.author._id;

    const handleLoadReplies = async () => {
        if (replies.length > 0) {
            setReplies([]);
            return;
        }
        setLoadingReplies(true);
        try {
            const response = await API.get(`/comments/${comment._id}/replies`);
            setReplies(response.data);
        } catch (error) {
            toast.error("Failed to load replies.");
        } finally {
            setLoadingReplies(false);
        }
    };
    
    const handleReplySubmit = async (e) => {
        e.preventDefault();
        if (!replyText.trim()) return;
        try {
            const response = await API.post(`/videos/${videoId}/comments`, {
                text: replyText,
                parentCommentId: comment._id,
            });
            setReplies([...replies, response.data]);
            if (onReplySubmitted) onReplySubmitted();
            setReplyText('');
            setShowReplyForm(false);
            toast.success("Reply posted!");
        } catch (error) {
            toast.error("Failed to post reply.");
        }
    };

    const handleUpdateSubmit = async (e) => {
        e.preventDefault();
        try {
            const response = await API.put(`/comments/${comment._id}`, { text: editText });
            if (onCommentUpdated) onCommentUpdated(response.data);
            setIsEditing(false);
            toast.success("Comment updated!");
        } catch (error) {
            toast.error("Failed to update comment.");
        }
    };
    
    const handleDelete = async () => {
        if(window.confirm("Are you sure you want to delete this comment? All replies will also be removed.")){
            try {
                await API.delete(`/comments/${comment._id}`);
                if (onCommentDeleted) onCommentDeleted(comment);
                toast.success("Comment deleted.");
            } catch (error) {
                toast.error("Failed to delete comment.");
            }
        }
    };

    return (
        <div className="flex space-x-3">
            <Link href={`/profile/${comment.author.username}`} className="flex-shrink-0 mt-1">
                <Avatar className="w-9 h-9">
                    <AvatarImage src={getAvatarUrl(comment.author?.username, comment.author?.avatar)} alt={comment.author?.username} />
                    <AvatarFallback>{comment.author?.username?.charAt(0) || "U"}</AvatarFallback>
                </Avatar>
            </Link>
            <div className="flex-1 min-w-0">
                <div className="flex justify-between items-center">
                    <p className="font-semibold text-sm text-foreground">
                        <Link href={`/profile/${comment.author.username}`} className="hover:text-primary transition-colors">{comment.author.username}</Link>{" "}
                        <span className="text-xs text-muted-foreground font-normal">{new Date(comment.createdAt).toLocaleString()}</span>
                    </p>
                    {isOwner && !isEditing && (
                        <div className="flex items-center space-x-2">
                             <Button variant="ghost" size="sm" onClick={() => setIsEditing(true)} className="h-6 px-2 text-xs text-muted-foreground hover:text-foreground">Edit</Button>
                             <Button variant="ghost" size="sm" onClick={handleDelete} className="h-6 px-2 text-xs text-destructive hover:text-destructive hover:bg-destructive/10">Delete</Button>
                        </div>
                    )}
                </div>

                {isEditing ? (
                    <form onSubmit={handleUpdateSubmit} className="mt-2">
                        <Textarea 
                            value={editText} 
                            onChange={(e) => setEditText(e.target.value)}
                            className="w-full text-sm resize-none"
                            rows={2}
                        />
                        <div className="flex space-x-2 mt-2">
                            <Button size="sm" type="submit" className="rounded-full px-4 h-7 text-xs">Save</Button>
                            <Button size="sm" variant="secondary" type="button" onClick={() => setIsEditing(false)} className="rounded-full px-4 h-7 text-xs">Cancel</Button>
                        </div>
                    </form>
                ) : (
                    <p className="text-foreground text-sm mt-1">{comment.text}</p>
                )}
                
                <div className="flex items-center space-x-4 text-xs mt-2">
                    <Button variant="link" onClick={() => setShowReplyForm(!showReplyForm)} className="h-auto p-0 font-semibold text-muted-foreground hover:text-foreground">Reply</Button>
                    <Button variant="link" onClick={handleLoadReplies} className="h-auto p-0 font-semibold text-muted-foreground hover:text-foreground">
                        {loadingReplies ? 'Loading...' : replies.length > 0 ? 'Hide Replies' : 'View Replies'}
                    </Button>
                </div>
                
                {showReplyForm && isAuthenticated && (
                    <form onSubmit={handleReplySubmit} className="mt-3">
                        <Textarea 
                            value={replyText} 
                            onChange={(e) => setReplyText(e.target.value)}
                            placeholder={`Replying to ${comment.author.username}...`}
                            className="w-full text-sm resize-none"
                            rows={2}
                        />
                        <div className="flex space-x-2 mt-2">
                             <Button size="sm" type="submit" className="rounded-full px-4 h-7 text-xs">Post Reply</Button>
                             <Button size="sm" variant="secondary" type="button" onClick={() => setShowReplyForm(false)} className="rounded-full px-4 h-7 text-xs">Cancel</Button>
                        </div>
                    </form>
                )}

                <div className="mt-4 space-y-4 pl-4 border-l-2 border-border ml-2">
                    {replies.map(reply => (
                        <Comment 
                            key={reply._id} 
                            comment={reply} 
                            videoId={videoId}
                            onCommentDeleted={() => setReplies(prev => prev.filter(r => r._id !== reply._id))}
                            onCommentUpdated={(updatedReply) => {
                                setReplies(prevReplies => prevReplies.map(r => r._id === updatedReply._id ? updatedReply : r));
                            }}
                            onReplySubmitted={onReplySubmitted} 
                        />
                    ))}
                </div>
            </div>
        </div>
    );
};

export default Comment;