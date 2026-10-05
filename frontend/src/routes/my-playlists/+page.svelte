<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { ListVideo, Plus, Trash2, Edit } from 'lucide-svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';

	const queryClient = useQueryClient();

	let isCreating = $state(false);
	let newPlaylistName = $state('');

	const playlistsQuery = createQuery(() => ({
		queryKey: ['playlists'],
		queryFn: async () => {
			const res = await fetchApi('/playlists');
			return res.playlists || res.data || [];
		}
	}), () => queryClient);

	const createPlaylistMutation = createMutation(() => ({
		mutationFn: async (title: string) => {
			return await fetchApi('/playlists', {
				method: 'POST',
				body: JSON.stringify({ title, description: '', visibility: 'public' })
			});
		},
		onSuccess: () => {
			isCreating = false;
			newPlaylistName = '';
			queryClient.invalidateQueries({ queryKey: ['playlists'] });
		}
	}), () => queryClient);

	const deletePlaylistMutation = createMutation(() => ({
		mutationFn: async (id: string) => {
			return await fetchApi(`/playlists/${id}`, { method: 'DELETE' });
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['playlists'] });
		}
	}), () => queryClient);

	function handleCreate() {
		if (!newPlaylistName.trim()) return;
		createPlaylistMutation.mutate(newPlaylistName.trim());
	}

	function handleDelete(e: Event, id: string) {
		e.preventDefault();
		e.stopPropagation();
		if (confirm('Are you sure you want to delete this playlist?')) {
			deletePlaylistMutation.mutate(id);
		}
	}
</script>

<svelte:head>
	<title>My Playlists - Aurahub</title>
</svelte:head>

<div class="space-y-4 md:space-y-6 px-4 md:px-0">
	<div class="flex items-center justify-between border-b pb-4 pt-2 md:pt-0">
		<div class="flex items-center gap-3">
			<ListVideo class="h-6 w-6 text-primary" />
			<h1 class="text-xl md:text-2xl font-bold tracking-tight">My Playlists</h1>
		</div>
		<Button 
			size="sm" 
			class="rounded-full"
			onclick={() => (isCreating = !isCreating)}
		>
			<Plus class="mr-2 h-4 w-4" /> New Playlist
		</Button>
	</div>

	{#if isCreating}
		<div class="bg-card rounded-xl border p-4 shadow-sm animate-in slide-in-from-top-2">
			<h3 class="mb-3 font-semibold">Create New Playlist</h3>
			<div class="flex flex-col sm:flex-row gap-3">
				<Input
					type="text"
					bind:value={newPlaylistName}
					placeholder="Playlist title"
					class="flex-1"
					disabled={createPlaylistMutation.isPending}
				/>
				<div class="flex gap-2 justify-end">
					<Button 
						variant="outline" 
						onclick={() => (isCreating = false)}
						disabled={createPlaylistMutation.isPending}
					>
						Cancel
					</Button>
					<Button 
						onclick={handleCreate}
						disabled={!newPlaylistName.trim() || createPlaylistMutation.isPending}
					>
						Create
					</Button>
				</div>
			</div>
		</div>
	{/if}

	{#if playlistsQuery.isPending}
		<div class="grid grid-cols-1 gap-4 sm:gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(4) as _}
				<div class="flex animate-pulse flex-col space-y-3">
					<Skeleton class="aspect-video w-full rounded-xl" />
					<Skeleton class="h-5 w-3/4 rounded" />
					<Skeleton class="h-4 w-1/2 rounded" />
				</div>
			{/each}
		</div>
	{:else if playlistsQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-xl bg-destructive/10 p-4 text-destructive">
				<p>{playlistsQuery.error.message || 'Failed to load playlists.'}</p>
				<Button variant="outline" class="mt-4" onclick={() => playlistsQuery.refetch()}>Try again</Button>
			</div>
		</div>
	{:else if !playlistsQuery.data || playlistsQuery.data.length === 0}
		<div
			class="border-muted-foreground/20 mx-4 md:mx-0 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-16 md:py-24 text-center"
		>
			<ListVideo class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-lg md:text-xl font-semibold">No playlists yet</h2>
			<p class="text-muted-foreground max-w-sm text-sm md:text-base px-4">
				Create a playlist to organize and share your favorite videos.
			</p>
			<Button variant="default" class="mt-6 rounded-full" onclick={() => (isCreating = true)}>
				<Plus class="mr-2 h-4 w-4" /> Create Playlist
			</Button>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-4 sm:gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each playlistsQuery.data as playlist}
				<a
					href={`/playlist/${playlist.id}`}
					class="group bg-card hover:bg-muted/50 flex flex-col overflow-hidden rounded-xl border shadow-sm transition-colors"
				>
					<div class="bg-muted relative aspect-video w-full overflow-hidden">
						{#if playlist.thumbnail}
							<img
								src={playlist.thumbnail}
								alt={playlist.title}
								class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
							/>
						{:else}
							<div class="flex h-full w-full items-center justify-center">
								<ListVideo class="text-muted-foreground/30 h-12 w-12" />
							</div>
						{/if}
						<div
							class="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 backdrop-blur-[2px] transition-opacity group-hover:opacity-100"
						>
							<span class="flex items-center gap-2 font-medium text-white drop-shadow-md">
								<ListVideo class="h-5 w-5" /> View Full Playlist
							</span>
						</div>
					</div>
					<div class="flex flex-1 flex-col p-3 md:p-4 relative">
						<h3 class="line-clamp-1 text-base md:text-lg font-semibold transition-colors group-hover:text-primary">
							{playlist.title}
						</h3>
						<p class="text-muted-foreground mt-1 text-xs md:text-sm">
							{playlist.videoCount || playlist.videos?.length || 0} videos
						</p>
						<div class="mt-2 text-[11px] md:text-xs text-muted-foreground">
							{playlist.visibility === 'private' ? 'Private' : 'Public'}
						</div>
						
						<div class="absolute top-3 right-3 opacity-100 md:opacity-0 transition-opacity group-hover:opacity-100 flex gap-1">
							<Button
								variant="ghost"
								size="icon"
								class="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
								onclick={(e) => handleDelete(e, playlist.id)}
								title="Delete Playlist"
							>
								<Trash2 class="h-4 w-4" />
							</Button>
						</div>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
