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

	const playlistsQuery = createQuery(
		() => ({
			queryKey: ['playlists'],
			queryFn: async () => {
				const res = await fetchApi('/playlists');
				return res.playlists || res.data || [];
			}
		}),
		() => queryClient
	);

	const createPlaylistMutation = createMutation(
		() => ({
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
		}),
		() => queryClient
	);

	const deletePlaylistMutation = createMutation(
		() => ({
			mutationFn: async (id: string) => {
				return await fetchApi(`/playlists/${id}`, { method: 'DELETE' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['playlists'] });
			}
		}),
		() => queryClient
	);

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

<div class="space-y-4 px-4 md:space-y-6 md:px-0">
	<div class="flex items-center justify-between border-b border-slate-border/60 pt-2 pb-4 md:pt-0">
		<div class="flex items-center gap-3">
			<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-aura-lavender/10 text-aura-lavender">
				<ListVideo class="h-5 w-5" />
			</div>
			<div>
				<h1 class="text-xl font-bold tracking-tight text-soft-ivory md:text-2xl font-serif">My Playlists</h1>
				<p class="text-xs text-muted-silver md:text-sm">Organize and curate your favorite content collections</p>
			</div>
		</div>
		<Button size="sm" class="rounded-full bg-aura-lavender text-midnight-ink font-semibold hover:bg-aura-lavender/90 shadow-sm" onclick={() => (isCreating = !isCreating)}>
			<Plus class="mr-1.5 h-4 w-4" /> New Playlist
		</Button>
	</div>

	{#if isCreating}
		<div class="animate-in slide-in-from-top-2 rounded-2xl border border-slate-border/80 bg-deep-slate p-5 shadow-xl">
			<h3 class="mb-3 font-semibold text-soft-ivory">Create New Playlist</h3>
			<div class="flex flex-col gap-3 sm:flex-row">
				<Input
					type="text"
					bind:value={newPlaylistName}
					placeholder="Playlist title"
					class="flex-1 rounded-xl border-slate-border/80 bg-midnight-ink/60 text-soft-ivory placeholder:text-muted-silver/50 focus-visible:ring-aura-lavender"
					disabled={createPlaylistMutation.isPending}
				/>
				<div class="flex justify-end gap-2">
					<Button
						variant="outline"
						class="rounded-xl border-slate-border text-muted-silver hover:bg-slate-elevated hover:text-soft-ivory"
						onclick={() => (isCreating = false)}
						disabled={createPlaylistMutation.isPending}
					>
						Cancel
					</Button>
					<Button
						class="rounded-xl bg-aura-lavender text-midnight-ink font-semibold hover:bg-aura-lavender/90"
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
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(4) as _}
				<div class="flex animate-pulse flex-col space-y-3 rounded-2xl border border-slate-border/30 bg-deep-slate/30 p-3">
					<Skeleton class="aspect-video w-full rounded-xl bg-slate-elevated" />
					<Skeleton class="h-5 w-3/4 rounded-lg bg-slate-elevated" />
					<Skeleton class="h-4 w-1/2 rounded-lg bg-slate-elevated" />
				</div>
			{/each}
		</div>
	{:else if playlistsQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-2xl border border-destructive/30 bg-destructive/10 p-6 text-destructive">
				<p>{playlistsQuery.error.message || 'Failed to load playlists.'}</p>
				<Button variant="outline" class="mt-4 rounded-full border-destructive/40" onclick={() => playlistsQuery.refetch()}
					>Try again</Button
				>
			</div>
		</div>
	{:else if !playlistsQuery.data || playlistsQuery.data.length === 0}
		<div
			class="mx-4 flex flex-col items-center justify-center rounded-3xl border border-dashed border-slate-border/80 bg-deep-slate/20 py-16 text-center md:mx-0 md:py-24"
		>
			<div class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-deep-slate text-muted-silver">
				<ListVideo class="h-7 w-7 opacity-60" />
			</div>
			<h2 class="mb-2 text-lg font-bold text-soft-ivory md:text-xl">No playlists yet</h2>
			<p class="max-w-sm px-4 text-sm text-muted-silver md:text-base">
				Create a playlist to organize and share your favorite videos.
			</p>
			<Button variant="default" class="mt-6 rounded-full bg-aura-lavender text-midnight-ink font-semibold hover:bg-aura-lavender/90 shadow-md shadow-aura-lavender/20" onclick={() => (isCreating = true)}>
				<Plus class="mr-2 h-4 w-4" /> Create Playlist
			</Button>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3 xl:grid-cols-4">
			{#each playlistsQuery.data as playlist}
				<a
					href={`/playlist/${playlist.id}`}
					class="group relative flex flex-col overflow-hidden rounded-2xl border border-slate-border/40 bg-deep-slate/30 p-2.5 transition-all hover:border-aura-lavender/40 hover:bg-deep-slate/80 shadow-md min-w-0"
				>
					<div class="relative aspect-video w-full overflow-hidden rounded-xl bg-slate-elevated">
						{#if playlist.thumbnail}
							<img
								src={playlist.thumbnail}
								alt={playlist.title}
								class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
							/>
						{:else}
							<div class="flex h-full w-full items-center justify-center bg-gradient-to-br from-deep-slate to-slate-elevated">
								<ListVideo class="h-10 w-10 text-muted-silver/40" />
							</div>
						{/if}
						<div
							class="absolute inset-0 flex items-center justify-center bg-midnight-ink/60 opacity-0 backdrop-blur-sm transition-opacity group-hover:opacity-100"
						>
							<span class="flex items-center gap-2 text-xs font-semibold text-soft-ivory drop-shadow-md">
								<ListVideo class="h-4 w-4 text-aura-lavender" /> View Full Playlist
							</span>
						</div>
					</div>
					<div class="relative flex flex-1 flex-col p-2 pt-3 min-w-0 overflow-hidden">
						<h3
							class="line-clamp-1 text-base font-semibold text-soft-ivory transition-colors group-hover:text-aura-lavender md:text-lg break-words [overflow-wrap:anywhere]"
							title={playlist.title}
						>
							{playlist.title}
						</h3>
						<p class="mt-1 text-xs text-muted-silver md:text-sm">
							{playlist.videoCount || playlist.videos?.length || 0} videos
						</p>
						<div class="mt-2 flex items-center gap-2">
							<span class="rounded-full border border-slate-border/60 bg-slate-elevated px-2 py-0.5 text-[10px] font-semibold text-muted-silver">
								{playlist.visibility === 'private' ? 'Private' : 'Public'}
							</span>
						</div>

						<div
							class="absolute top-3 right-2 flex gap-1 opacity-100 transition-opacity group-hover:opacity-100 md:opacity-0"
						>
							<Button
								variant="ghost"
								size="icon"
								class="h-8 w-8 rounded-full text-muted-silver hover:bg-destructive/10 hover:text-destructive"
								onclick={(e: MouseEvent) => handleDelete(e, playlist.id)}
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
