<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { Plus, X, ListVideo, Loader2, Check } from 'lucide-svelte';
	import { userState } from '#lib/user.svelte';
	import { onMount } from 'svelte';
	import { fade, scale } from 'svelte/transition';

	let { videoId, isOpen = $bindable(false) } = $props<{ videoId: string; isOpen: boolean }>();

	let playlists: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');

	let isCreating = $state(false);
	let newPlaylistTitle = $state('');

	// State to track which playlists contain this video
	let savingStatus: Record<string, 'idle' | 'saving' | 'saved' | 'error'> = $state({});
	let containsVideo: Record<string, boolean> = $state({});

	$effect(() => {
		if (isOpen) {
			if (!userState.user) {
				alert('Please login to save to a playlist.');
				isOpen = false;
				return;
			}
			loadPlaylists();
		}
	});

	async function loadPlaylists() {
		isLoading = true;
		error = '';
		try {
			const res = await fetchApi('/playlists');
			playlists = res.playlists || res.data || [];

			// Check which playlists contain this video
			playlists.forEach(async (p) => {
				savingStatus[p.id] = 'idle';
				// Ideally, the backend would return this info in the /playlists list,
				// but if not, we can query the playlist details or just assume false
				// and let the toggle handle it. For a quick implementation, we will
				// assume false initially.
				containsVideo[p.id] = false;
			});
		} catch (err: any) {
			error = err.message || 'Failed to load playlists.';
		} finally {
			isLoading = false;
		}
	}

	async function handleToggle(playlist: any) {
		const pid = playlist.id;
		if (savingStatus[pid] === 'saving') return;
		savingStatus[pid] = 'saving';

		try {
			const action = containsVideo[pid] ? 'remove' : 'add';
			// Using the TogglePlaylistVideoHandler which is mapped to POST /playlists/:id/videos
			const res = await fetchApi(`/playlists/${pid}/videos`, {
				method: 'POST',
				body: JSON.stringify({ videoId, action })
			});
			containsVideo[pid] = action === 'add';
			savingStatus[pid] = 'saved';
			setTimeout(() => {
				if (savingStatus[pid] === 'saved') savingStatus[pid] = 'idle';
			}, 2000);
		} catch (err) {
			console.error(err);
			savingStatus[pid] = 'error';
			setTimeout(() => {
				if (savingStatus[pid] === 'error') savingStatus[pid] = 'idle';
			}, 2000);
		}
	}

	async function handleCreate() {
		if (!newPlaylistTitle.trim()) return;

		isCreating = true;
		try {
			const res = await fetchApi('/playlists', {
				method: 'POST',
				body: JSON.stringify({ title: newPlaylistTitle.trim(), isPublic: true })
			});

			const created = res.playlist;
			if (created) {
				playlists = [created, ...playlists];
				savingStatus[created.id] = 'idle';
				containsVideo[created.id] = false;

				// Automatically add to the new playlist
				await handleToggle(created);
			}
			newPlaylistTitle = '';
		} catch (err: any) {
			alert('Failed to create playlist');
		} finally {
			isCreating = false;
		}
	}
</script>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		transition:fade={{ duration: 200 }}
		role="dialog"
		aria-modal="true"
	>
		<div
			class="bg-card flex max-h-[85vh] w-full max-w-md flex-col overflow-hidden rounded-2xl border border-border shadow-2xl"
			transition:scale={{ duration: 200, start: 0.95 }}
		>
			<div class="bg-card sticky top-0 z-10 flex items-center justify-between border-b px-6 py-4">
				<h2 class="text-xl font-semibold tracking-tight">Save to playlist</h2>
				<button
					onclick={() => (isOpen = false)}
					class="hover:bg-muted text-muted-foreground hover:text-foreground -mr-2 rounded-full p-2 transition-colors"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<div class="custom-scrollbar flex-1 overflow-y-auto p-6">
				{#if isLoading}
					<div class="flex items-center justify-center py-8">
						<Loader2 class="h-8 w-8 animate-spin text-primary" />
					</div>
				{:else if error}
					<div class="py-8 text-center font-medium text-rose-500">{error}</div>
				{:else}
					<div class="mb-6 space-y-2">
						{#each playlists as playlist}
							<button
								onclick={() => handleToggle(playlist)}
								class="hover:bg-muted/50 group flex w-full items-center justify-between rounded-xl p-3 text-left transition-colors"
							>
								<div class="flex items-center gap-3">
									<div
										class={`flex h-5 w-5 items-center justify-center rounded border transition-colors ${containsVideo[playlist.id] ? 'border-primary bg-primary' : 'border-muted-foreground/40 group-hover:border-foreground/60'}`}
									>
										{#if containsVideo[playlist.id]}
											<Check class="text-primary-foreground h-3.5 w-3.5" />
										{/if}
									</div>
									<span class="max-w-[200px] truncate font-medium sm:max-w-[260px]"
										>{playlist.title}</span
									>
								</div>

								{#if savingStatus[playlist.id] === 'saving'}
									<Loader2 class="text-muted-foreground h-4 w-4 animate-spin" />
								{:else if savingStatus[playlist.id] === 'error'}
									<span class="text-xs font-medium text-rose-500">Failed</span>
								{/if}
							</button>
						{:else}
							<div class="text-muted-foreground flex flex-col items-center py-6 text-center">
								<ListVideo class="mb-2 h-8 w-8 opacity-50" />
								<p>You don't have any playlists yet.</p>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<div class="bg-muted/20 border-t p-6">
				<form
					onsubmit={(e) => {
						e.preventDefault();
						handleCreate();
					}}
					class="flex gap-2"
				>
					<input
						type="text"
						bind:value={newPlaylistTitle}
						placeholder="New playlist name..."
						class="flex-1 rounded-xl border border-border bg-background px-4 py-2 text-sm transition-shadow focus:ring-2 focus:ring-primary focus:outline-none"
						required
					/>
					<button
						type="submit"
						disabled={isCreating || !newPlaylistTitle.trim()}
						class="bg-foreground hover:bg-foreground/90 flex shrink-0 items-center gap-2 rounded-xl px-4 py-2 text-sm font-semibold text-background transition-colors disabled:opacity-50"
					>
						{#if isCreating}
							<Loader2 class="h-4 w-4 animate-spin" />
						{:else}
							<Plus class="h-4 w-4" /> Create
						{/if}
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
