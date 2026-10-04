"use client";

import React, { useState, useEffect } from 'react';
import API from '@/lib/api';
import { toast } from 'react-toastify';

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { ScrollArea } from "@/components/ui/scroll-area";

const SaveToPlaylistModal = ({ videoId, onClose }) => {
    const [playlists, setPlaylists] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showCreateForm, setShowCreateForm] = useState(false);
    const [newPlaylistTitle, setNewPlaylistTitle] = useState("");
    const [newPlaylistIsPublic, setNewPlaylistIsPublic] = useState(true);

    useEffect(() => {
        const fetchPlaylists = async () => {
            try {
                const response = await API.get('/playlists');
                setPlaylists(response.data);
            } catch (error) {
                console.error("Failed to fetch playlists:", error);
            } finally {
                setLoading(false);
            }
        };
        fetchPlaylists();
    }, []);

    const handleToggleVideoInPlaylist = async (playlistId) => {
        try {
            const response = await API.put('/playlists', { playlistId, videoId });
            setPlaylists(prev => prev.map(p => p._id === playlistId ? response.data : p));
        } catch (error) {
            toast.error("Failed to update playlist.");
        }
    };
    
    const handleCreatePlaylist = async (e) => {
        e.preventDefault();
        if (!newPlaylistTitle.trim()) return;
        try {
            const response = await API.post('/playlists', { 
                title: newPlaylistTitle, 
                isPublic: newPlaylistIsPublic 
            });
            setPlaylists([...playlists, response.data]);
            setNewPlaylistTitle("");
            setShowCreateForm(false);
            toast.success("Playlist created!");
        } catch (error) {
            toast.error("Failed to create playlist.");
        }
    };

    return (
        <Dialog open={true} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="sm:max-w-sm">
                <DialogHeader>
                    <DialogTitle>Save to...</DialogTitle>
                </DialogHeader>
                
                {loading ? (
                    <p className="text-muted-foreground py-4 text-sm">Loading playlists...</p>
                ) : (
                    <ScrollArea className="max-h-60 pr-4 mt-2">
                        <ul className="space-y-2">
                            {playlists.map(playlist => (
                                <li key={playlist._id}>
                                    <Label className="flex items-center space-x-3 cursor-pointer p-2 rounded-lg hover:bg-muted transition-colors font-medium">
                                        <Checkbox 
                                            checked={playlist.videos.includes(videoId)}
                                            onCheckedChange={() => handleToggleVideoInPlaylist(playlist._id)}
                                        />
                                        <span>{playlist.title}</span>
                                    </Label>
                                </li>
                            ))}
                        </ul>
                    </ScrollArea>
                )}
                
                <div className="mt-4 pt-4 border-t border-border">
                    {showCreateForm ? (
                        <form onSubmit={handleCreatePlaylist} className="animate-in fade-in slide-in-from-top-2 duration-200 space-y-4">
                            <Input 
                                type="text"
                                placeholder="Enter playlist name..."
                                value={newPlaylistTitle}
                                onChange={(e) => setNewPlaylistTitle(e.target.value)}
                                required
                            />
                            <div className="flex items-center justify-between">
                                <Label htmlFor="isPublic" className="flex items-center gap-2 cursor-pointer font-medium">
                                    <Checkbox 
                                        id="isPublic"
                                        checked={newPlaylistIsPublic}
                                        onCheckedChange={(checked) => setNewPlaylistIsPublic(checked)}
                                    />
                                    Public
                                </Label>
                                <div className="flex justify-end space-x-2">
                                    <Button variant="ghost" size="sm" type="button" onClick={() => setShowCreateForm(false)}>
                                        Cancel
                                    </Button>
                                    <Button size="sm" type="submit">
                                        Create
                                    </Button>
                                </div>
                            </div>
                        </form>
                    ) : (
                        <Button 
                            variant="ghost" 
                            className="w-full justify-start text-primary hover:text-primary hover:bg-primary/10"
                            onClick={() => setShowCreateForm(true)}
                        >
                            + Create new playlist
                        </Button>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
};

export default SaveToPlaylistModal;