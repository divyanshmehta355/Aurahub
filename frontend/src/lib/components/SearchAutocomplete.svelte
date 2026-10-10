<script lang="ts">
	import { goto } from '$app/navigation';
	import { fetchApi } from '#lib/api';
	import { Search, Film, User as UserIcon, ArrowRight, ShieldAlert } from 'lucide-svelte';

	interface AutocompleteVideo {
		id: string;
		title: string;
		thumbnailUrl?: string;
		category?: string;
	}

	interface AutocompleteUser {
		id: string;
		username: string;
		avatar?: string;
	}

	let {
		autofocus = false,
		placeholder = 'Search videos...',
		class: className = '',
		onSearchSubmitted
	}: {
		autofocus?: boolean;
		placeholder?: string;
		class?: string;
		onSearchSubmitted?: () => void;
	} = $props();

	let query = $state('');
	let isOpen = $state(false);
	let isLoading = $state(false);
	let videos = $state<AutocompleteVideo[]>([]);
	let users = $state<AutocompleteUser[]>([]);
	let adultHidden = $state(false);
	let selectedIndex = $state(-1);
	let containerRef = $state<HTMLDivElement | null>(null);
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;

	let totalItems = $derived(videos.length + users.length + (query.trim() ? 1 : 0));

	function handleInput(e: Event) {
		const target = e.target as HTMLInputElement;
		query = target.value;
		selectedIndex = -1;

		if (debounceTimer) clearTimeout(debounceTimer);

		if (query.trim().length < 2) {
			videos = [];
			users = [];
			adultHidden = false;
			isOpen = false;
			return;
		}

		isLoading = true;
		debounceTimer = setTimeout(async () => {
			try {
				const res = await fetchApi(`/search/autocomplete?q=${encodeURIComponent(query.trim())}`);
				videos = res.videos || [];
				users = res.users || [];
				adultHidden = !!res.adultHidden;
				isOpen = videos.length > 0 || users.length > 0 || adultHidden;
			} catch (err) {
				console.error('Autocomplete fetch error:', err);
			} finally {
				isLoading = false;
			}
		}, 150);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!isOpen && e.key !== 'Enter') {
			if (e.key === 'ArrowDown' && query.trim().length >= 2) {
				isOpen = true;
			}
			return;
		}

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = (selectedIndex + 1) % totalItems;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = (selectedIndex - 1 + totalItems) % totalItems;
		} else if (e.key === 'Escape') {
			isOpen = false;
			selectedIndex = -1;
		} else if (e.key === 'Enter') {
			if (selectedIndex >= 0) {
				e.preventDefault();
				executeSelection(selectedIndex);
			} else {
				submitSearch();
			}
		}
	}

	function executeSelection(index: number) {
		if (index < videos.length) {
			const video = videos[index];
			goto(`/watch/${video.id}`);
			closeDropdown();
		} else if (index < videos.length + users.length) {
			const user = users[index - videos.length];
			goto(`/user/${user.username}`);
			closeDropdown();
		} else {
			submitSearch();
		}
	}

	function submitSearch() {
		if (!query.trim()) return;
		closeDropdown();
		goto(`/search?q=${encodeURIComponent(query.trim())}`);
		onSearchSubmitted?.();
	}

	function closeDropdown() {
		isOpen = false;
		selectedIndex = -1;
	}

	// Handle clicking outside to dismiss
	function handleDocumentClick(e: MouseEvent) {
		if (containerRef && !containerRef.contains(e.target as Node)) {
			closeDropdown();
		}
	}
</script>

<svelte:window onclick={handleDocumentClick} />

<div bind:this={containerRef} class="relative w-full {className}">
	<form
		onsubmit={(e) => {
			e.preventDefault();
			submitSearch();
		}}
		class="relative w-full"
	>
		<Search class="text-muted-foreground pointer-events-none absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2" />
		<input
			type="search"
			value={query}
			oninput={handleInput}
			onkeydown={handleKeydown}
			onfocus={() => {
				if (videos.length > 0 || users.length > 0 || adultHidden) isOpen = true;
			}}
			{placeholder}
			{autofocus}
			class="bg-muted/50 border-border focus:ring-primary/20 w-full rounded-full border py-2 pr-4 pl-10 text-sm transition-all focus:ring-2 focus:outline-none"
			autocomplete="off"
			spellcheck="false"
		/>
	</form>

	<!-- Autocomplete Dropdown -->
	{#if isOpen}
		<div
			class="bg-popover/95 border-border/80 absolute top-full z-50 mt-2 max-h-[480px] w-full overflow-hidden overflow-y-auto rounded-2xl border shadow-2xl backdrop-blur-md"
		>
			<!-- Video suggestions -->
			{#if videos.length > 0}
				<div class="px-2 pt-2 pb-1">
					<div class="text-muted-foreground flex items-center gap-1.5 px-3 py-1 text-[11px] font-semibold tracking-wider uppercase">
						<Film class="h-3 w-3" />
						<span>Videos</span>
					</div>
					{#each videos as video, idx}
						<button
							type="button"
							class="hover:bg-accent/80 flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left transition-colors {selectedIndex === idx ? 'bg-accent text-accent-foreground' : ''}"
							onclick={() => {
								goto(`/watch/${video.id}`);
								closeDropdown();
							}}
						>
							{#if video.thumbnailUrl}
								<img
									src={video.thumbnailUrl}
									alt=""
									class="bg-muted h-9 w-16 shrink-0 rounded-md object-cover"
								/>
							{:else}
								<div class="bg-muted text-muted-foreground flex h-9 w-16 shrink-0 items-center justify-center rounded-md text-xs">
									<Film class="h-4 w-4 opacity-40" />
								</div>
							{/if}
							<div class="min-w-0 flex-1">
								<p class="truncate text-xs font-medium md:text-sm">{video.title}</p>
								{#if video.category}
									<span class="text-muted-foreground text-[10px]">{video.category}</span>
								{/if}
							</div>
						</button>
					{/each}
				</div>
			{/if}

			<!-- User suggestions -->
			{#if users.length > 0}
				<div class="border-border/60 border-t px-2 pt-2 pb-1">
					<div class="text-muted-foreground flex items-center gap-1.5 px-3 py-1 text-[11px] font-semibold tracking-wider uppercase">
						<UserIcon class="h-3 w-3" />
						<span>Creators</span>
					</div>
					{#each users as user, idx}
						{@const userIdx = videos.length + idx}
						<button
							type="button"
							class="hover:bg-accent/80 flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left transition-colors {selectedIndex === userIdx ? 'bg-accent text-accent-foreground' : ''}"
							onclick={() => {
								goto(`/user/${user.username}`);
								closeDropdown();
							}}
						>
							<img
								src={user.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${user.username}`}
								alt={user.username}
								class="bg-muted h-7 w-7 shrink-0 rounded-full object-cover"
							/>
							<div class="min-w-0 flex-1">
								<p class="truncate text-xs font-medium md:text-sm">{user.username}</p>
							</div>
						</button>
					{/each}
				</div>
			{/if}

			<!-- 18+ content hidden hint -->
			{#if adultHidden}
				<div class="bg-amber-500/10 border-border/40 text-amber-300 flex items-center gap-2 border-t px-4 py-2 text-[11px]">
					<ShieldAlert class="h-3.5 w-3.5 shrink-0 text-amber-400" />
					<span>18+ results available with adult filter enabled.</span>
				</div>
			{/if}

			<!-- Full search button row -->
			{#if query.trim()}
				{@const searchRowIdx = videos.length + users.length}
				<div class="border-border/60 border-t p-1.5">
					<button
						type="button"
						class="hover:bg-accent/80 text-muted-foreground hover:text-foreground flex w-full items-center justify-between rounded-xl px-3 py-2 text-left text-xs font-medium transition-colors {selectedIndex === searchRowIdx ? 'bg-accent text-accent-foreground' : ''}"
						onclick={submitSearch}
					>
						<span class="flex items-center gap-2 truncate">
							<Search class="h-3.5 w-3.5 shrink-0" />
							Search for "{query.trim()}"
						</span>
						<ArrowRight class="h-3.5 w-3.5 shrink-0 opacity-60" />
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
