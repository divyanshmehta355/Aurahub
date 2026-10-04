<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { Trash2, Edit2, Check, X, Play, Share2, Globe, Lock } from 'lucide-svelte';
	import { userState } from '#lib/user.svelte';

	let id = $derived(page.params.id);
	let playlist: any = $state(null);
	let isLoading = $state(true);
	let error = $state('');

	let isEditing = $state(false);
	let editName = $state('');

	async function loadPlaylist() {
		isLoading = true;
		error = '';
		try {
			const res = await fetchApi(`/playlists/${id}`);
			playlist = res.playlist || res;
			if (res.videos) {
				playlist.videos = res.videos;
			}
			editName = playlist.title;
		} catch (err: any) {
			error = err.message || 'Failed to load playlist.';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (id) {
			loadPlaylist();
		}
	});

	async function saveEdit() {
		if (!editName.trim()) return;
		try {
			await fetchApi(`/playlists/${id}`, {
				method: 'PUT',
				body: JSON.stringify({ title: editName.trim() })
			});
			playlist.title = editName.trim();
			isEditing = false;
		} catch (err: any) {
			alert('Failed to rename playlist');
		}
	}

	async function togglePrivacy() {
		try {
			await fetchApi(`/playlists/${id}`, {
				method: 'PUT',
				body: JSON.stringify({ isPublic: !playlist.isPublic, title: playlist.title })
			});
			playlist.isPublic = !playlist.isPublic;
		} catch (err: any) {
			alert('Failed to update privacy');
		}
	}

	function handleShare() {
		navigator.clipboard.writeText(window.location.href);
		alert('Link copied to clipboard!');
	}

	async function removeVideo(videoId: string | number) {
		try {
			await fetchApi(`/playlists/${id}/videos`, {
				method: 'POST',
				body: JSON.stringify({ videoId, action: 'remove' })
			});
			// Toggle implies removal if it was already in
			playlist.videos = playlist.videos.filter((v: any) => v.id !== videoId && v.fileId !== videoId);
		} catch (err) {
			alert('Failed to remove video');
		}
	}

	function formatDuration(seconds: number) {
		if (!seconds) return '0:00';
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
</script>

<svelte:head>
	<title>{playlist ? playlist.name : 'Playlist'} - Aurahub</title>
</svelte:head>

<div class="max-w-[1200px] mx-auto py-6">
	{#if isLoading}
		<div class="flex gap-8 animate-pulse">
			<div class="w-1/3 bg-muted h-[400px] rounded-xl"></div>
			<div class="w-2/3 space-y-4">
				<div class="h-24 bg-muted rounded"></div>
				<div class="h-24 bg-muted rounded"></div>
				<div class="h-24 bg-muted rounded"></div>
			</div>
		</div>
	{:else if error || !playlist}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-xl bg-red-100 p-4 text-red-600 dark:bg-red-900/30 dark:text-red-400">
				<p>{error || 'Playlist not found.'}</p>
			</div>
		</div>
	{:else}
		<div class="flex flex-col md:flex-row gap-8">
			<!-- Playlist Info Sidebar -->
			<div class="w-full md:w-[350px] shrink-0">
				<div class="bg-card border rounded-2xl p-6 sticky top-24 shadow-sm">
					<!-- Thumbnail Stack effect -->
					<div class="aspect-video bg-muted rounded-xl mb-6 relative overflow-hidden flex items-center justify-center group shadow-md">
						{#if playlist.videos && playlist.videos.length > 0}
							<img 
								src={playlist.videos[0].thumbnailUrl || `https://api.dicebear.com/7.x/identicon/svg?seed=${playlist.videos[0].title}`} 
								alt="Cover" 
								class="w-full h-full object-cover opacity-50 blur-sm scale-110" 
							/>
							<img 
								src={playlist.videos[0].thumbnailUrl || `https://api.dicebear.com/7.x/identicon/svg?seed=${playlist.videos[0].title}`} 
								alt="Cover" 
								class="w-4/5 h-full object-cover absolute shadow-lg group-hover:scale-105 transition-transform duration-300" 
							/>
						{:else}
							<div class="text-muted-foreground opacity-50 flex flex-col items-center">
								<Play class="w-12 h-12 mb-2" />
								<span>Empty</span>
							</div>
						{/if}
						<div class="absolute inset-0 bg-black/20 group-hover:bg-black/10 transition-colors pointer-events-none"></div>
					</div>

					{#if isEditing}
						<div class="space-y-3 mb-4">
							<input 
								type="text" 
								bind:value={editName} 
								class="w-full bg-background border rounded-lg px-3 py-2 font-bold text-xl"
								autofocus
							/>
							<div class="flex justify-end gap-2">
								<button onclick={() => {isEditing = false; editName = playlist.title}} class="p-2 text-muted-foreground hover:bg-muted rounded-lg">
									<X class="w-4 h-4" />
								</button>
								<button onclick={saveEdit} class="p-2 text-primary hover:bg-primary/10 rounded-lg">
									<Check class="w-4 h-4" />
								</button>
							</div>
						</div>
					{:else}
						<div class="flex items-start justify-between gap-4 mb-2">
							<h1 class="text-2xl font-bold tracking-tight break-words">{playlist.title}</h1>
							{#if userState.user?.id === playlist.ownerId}
								<button onclick={() => isEditing = true} class="p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground rounded-md transition-colors">
									<Edit2 class="w-4 h-4" />
								</button>
							{/if}
						</div>
					{/if}

					<div class="text-sm text-muted-foreground space-y-1 mb-6 font-medium">
						<p>{playlist.user?.username || 'Unknown User'}</p>
						<p>{playlist.videos?.length || 0} videos</p>
					</div>

					{#if playlist.videos && playlist.videos.length > 0}
						<a 
							href={`/watch/${playlist.videos[0].fileId || playlist.videos[0].id}?list=${playlist.id}`}
							class="w-full inline-flex items-center justify-center rounded-full bg-primary text-primary-foreground hover:bg-primary/90 h-12 px-8 font-semibold transition-colors shadow-sm"
						>
							<Play class="w-5 h-5 mr-2 fill-current" />
							Play all
						</a>
					{/if}

					<div class="flex gap-2 mt-4 w-full">
						<button 
							onclick={handleShare}
							class="flex-1 inline-flex items-center justify-center rounded-full bg-secondary text-secondary-foreground hover:bg-secondary/80 h-10 px-4 font-semibold transition-colors shadow-sm"
						>
							<Share2 class="w-4 h-4 mr-2" />
							Share
						</button>
						
						{#if userState.user?.id === playlist.ownerId}
							<button 
								onclick={togglePrivacy}
								class="flex-1 inline-flex items-center justify-center rounded-full bg-secondary text-secondary-foreground hover:bg-secondary/80 h-10 px-4 font-semibold transition-colors shadow-sm"
							>
								{#if playlist.isPublic}
									<Globe class="w-4 h-4 mr-2" /> Public
								{:else}
									<Lock class="w-4 h-4 mr-2" /> Private
								{/if}
							</button>
						{/if}
					</div>
				</div>
			</div>

			<!-- Videos List -->
			<div class="flex-1">
				{#if !playlist.videos || playlist.videos.length === 0}
					<div class="bg-card border rounded-2xl p-12 text-center shadow-sm">
						<p class="text-lg font-medium text-muted-foreground">No videos in this playlist yet.</p>
					</div>
				{:else}
					<div class="flex flex-col gap-3">
						{#each playlist.videos as video, i}
							<div class="group relative flex items-center gap-4 hover:bg-muted/50 p-3 rounded-xl transition-colors border border-transparent hover:border-border">
								<span class="text-muted-foreground font-medium w-6 text-center">{i + 1}</span>
								
								<a href={`/watch/${video.fileId || video.id}?list=${playlist.id}`} class="flex-1 flex flex-col sm:flex-row gap-4">
									<div class="w-full sm:w-40 shrink-0">
										<VideoThumbnail 
											{video} 
											class="aspect-video rounded-lg"
										>
											<div class="absolute right-1.5 bottom-1.5 rounded bg-black/80 px-1.5 py-0.5 text-[10px] font-medium text-white backdrop-blur-sm">
												{formatDuration(video.duration)}
											</div>
										</VideoThumbnail>
									</div>
									<div class="flex flex-col justify-center py-1">
										<h3 class="text-base font-semibold leading-tight line-clamp-2 group-hover:text-primary transition-colors">
											{video.title}
										</h3>
										<p class="text-sm text-muted-foreground mt-1">
											{video.uploader_username || 'Unknown'} • {video.views || 0} views
										</p>
									</div>
								</a>

								{#if userState.user?.id === playlist.userId}
									<button 
										onclick={() => removeVideo(video.id || video.fileId)}
										class="p-2 text-muted-foreground hover:text-red-500 hover:bg-red-500/10 rounded-full opacity-0 group-hover:opacity-100 transition-all shrink-0"
										title="Remove from playlist"
									>
										<Trash2 class="w-5 h-5" />
									</button>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
