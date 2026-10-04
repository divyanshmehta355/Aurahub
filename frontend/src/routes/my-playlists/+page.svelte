<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { ListVideo, Plus, Trash2, Edit } from 'lucide-svelte';

	let playlists: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');
	
	let isCreating = $state(false);
	let newPlaylistName = $state('');

	async function loadPlaylists() {
		isLoading = true;
		error = '';
		try {
			const res = await fetchApi('/playlists');
			playlists = res.playlists || res.data || [];
			
			playlists.forEach(async (playlist, index) => {
				try {
					const details = await fetchApi(`/playlists/${playlist.id}`);
					if (details && details.videos) {
						playlists[index] = { ...playlist, videoCount: details.videos.length };
					}
				} catch (e) {
					// Ignore
				}
			});
		} catch (err: any) {
			error = err.message || 'Failed to load playlists.';
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadPlaylists();
	});

	async function handleCreatePlaylist(e: Event) {
		e.preventDefault();
		if (!newPlaylistName.trim()) return;

		try {
			const res = await fetchApi('/playlists', {
				method: 'POST',
				body: JSON.stringify({ title: newPlaylistName.trim() })
			});
			
			// Refresh list or optimistic update
			if (res.playlist) {
				playlists = [res.playlist, ...playlists];
			} else {
				await loadPlaylists();
			}
			
			newPlaylistName = '';
			isCreating = false;
		} catch (err: any) {
			alert('Failed to create playlist');
		}
	}

	async function handleDelete(id: string | number) {
		if (!confirm('Are you sure you want to delete this playlist?')) return;
		try {
			await fetchApi(`/playlists/${id}`, { method: 'DELETE' });
			playlists = playlists.filter(p => p.id !== id);
		} catch (err) {
			alert('Failed to delete playlist');
		}
	}
</script>

<svelte:head>
	<title>My Playlists - Aurahub</title>
</svelte:head>

<div class="space-y-6 max-w-6xl mx-auto">
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b">
		<h1 class="text-2xl font-bold tracking-tight">My Playlists</h1>
		<button 
			onclick={() => isCreating = !isCreating}
			class="inline-flex items-center justify-center rounded-full bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2 text-sm font-medium transition-colors"
		>
			<Plus class="w-4 h-4 mr-2" />
			New Playlist
		</button>
	</div>

	{#if isCreating}
		<div class="bg-card border rounded-xl p-6 shadow-sm mb-6 max-w-md animate-in slide-in-from-top-2">
			<h3 class="font-semibold text-lg mb-4">Create New Playlist</h3>
			<form onsubmit={handleCreatePlaylist} class="space-y-4">
				<div>
					<label for="name" class="block text-sm font-medium mb-1">Name</label>
					<input 
						id="name" 
						type="text" 
						bind:value={newPlaylistName} 
						placeholder="e.g., Summer Vibes 2024"
						class="w-full bg-background border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
						required
					/>
				</div>
				<div class="flex justify-end gap-2">
					<button 
						type="button" 
						onclick={() => {isCreating = false; newPlaylistName = '';}}
						class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors"
					>
						Cancel
					</button>
					<button 
						type="submit" 
						class="px-4 py-2 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
						disabled={!newPlaylistName.trim()}
					>
						Create
					</button>
				</div>
			</form>
		</div>
	{/if}

	{#if isLoading}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(4) as _}
				<div class="bg-card border rounded-xl overflow-hidden animate-pulse">
					<div class="aspect-video bg-muted w-full"></div>
					<div class="p-4 space-y-3">
						<div class="h-5 bg-muted rounded w-3/4"></div>
						<div class="h-4 bg-muted rounded w-1/2"></div>
					</div>
				</div>
			{/each}
		</div>
	{:else if error}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-xl bg-red-100 p-4 text-red-600 dark:bg-red-900/30 dark:text-red-400">
				<p>{error}</p>
			</div>
		</div>
	{:else if playlists.length === 0}
		<div class="border-muted-foreground/20 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-24 text-center">
			<ListVideo class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-xl font-semibold">No playlists yet</h2>
			<p class="text-muted-foreground max-w-sm">
				Create a playlist to organize your favorite videos.
			</p>
			<button 
				onclick={() => isCreating = true}
				class="mt-6 text-primary hover:underline font-medium"
			>
				Create one now
			</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each playlists as playlist}
				<div class="group relative bg-card border rounded-xl overflow-hidden hover:shadow-md transition-all">
					<a href={`/playlist/${playlist.id}`} class="block aspect-video bg-muted relative overflow-hidden">
						<!-- Display a grid of thumbnails if there are videos, or a placeholder -->
						<div class="absolute inset-0 flex items-center justify-center bg-secondary">
							<ListVideo class="w-12 h-12 text-muted-foreground/50" />
						</div>
						
						<div class="absolute inset-0 bg-black/40 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
							<div class="flex items-center gap-2 text-white font-medium">
								<ListVideo class="w-5 h-5" />
								<span>Play All</span>
							</div>
						</div>
					</a>
					<div class="p-4 flex items-start justify-between">
						<div>
							<a href={`/playlist/${playlist.id}`} class="font-semibold text-lg line-clamp-1 hover:text-primary transition-colors">
								{playlist.title}
							</a>
							<p class="text-sm text-muted-foreground mt-1">
								{playlist.videoCount || 0} videos
							</p>
						</div>
						<button 
							onclick={() => handleDelete(playlist.id)}
							class="text-muted-foreground hover:text-red-500 transition-colors p-1"
							title="Delete Playlist"
						>
							<Trash2 class="w-4 h-4" />
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
