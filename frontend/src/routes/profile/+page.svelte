<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { userState } from '#lib/user.svelte';
	import { fetchApi } from '#lib/api';
	import { Image as ImageIcon, Shield, User, CheckCircle2, XCircle, Loader2 } from 'lucide-svelte';

	let activeTab = $state('profile'); // 'profile' | 'security'

	// Profile State
	let username = $state(userState.user?.username || '');
	let bio = $state(userState.user?.bio || '');
	let avatar = $state(userState.user?.avatar || '');
	let banner = $state(userState.user?.banner || '');

	// Security State
	let email = $state(userState.user?.email || '');
	let password = $state('');
	let confirmPassword = $state('');

	let isSubmitting = $state(false);
	let message = $state({ type: '', text: '' });

	$effect(() => {
		if (userState.user) {
			if (!username) username = userState.user.username || '';
			if (!bio) bio = userState.user.bio || '';
			if (!email) email = userState.user.email || '';
			if (!avatar) avatar = userState.user.avatar || '';
			if (!banner) banner = userState.user.banner || '';
		}
	});

	onMount(() => {
		if (!userState.user && !userState.isLoading) {
			goto('/login');
		}
	});

	async function handleImageUpload(e: Event, type: 'avatar' | 'banner') {
		const file = (e.target as HTMLInputElement).files?.[0];
		if (!file) return;

		const formData = new FormData();
		formData.append('avatar', file);

		try {
			const res = await fetchApi('/upload/avatar', {
				method: 'POST',
				body: formData
			});

			if (res.url) {
				if (type === 'avatar') avatar = res.url;
				if (type === 'banner') banner = res.url;
				
				// Immediately save
				await saveProfile();
			}
		} catch (err: any) {
			alert('Failed to upload image: ' + err.message);
		}
	}

	async function saveProfile(e?: Event) {
		if (e) e.preventDefault();
		isSubmitting = true;
		message = { type: '', text: '' };

		try {
			const payload = { username, bio, avatar, banner };
			const updatedUser = await fetchApi('/users/profile', {
				method: 'PUT',
				body: JSON.stringify(payload)
			});
			
			// Update local state
			userState.user = { ...userState.user, ...updatedUser };
			message = { type: 'success', text: 'Profile updated successfully!' };
		} catch (err: any) {
			message = { type: 'error', text: err.message || 'Failed to update profile.' };
		} finally {
			isSubmitting = false;
		}
	}

	async function saveSecurity(e: Event) {
		e.preventDefault();
		
		if (password && password !== confirmPassword) {
			message = { type: 'error', text: 'Passwords do not match.' };
			return;
		}

		isSubmitting = true;
		message = { type: '', text: '' };

		try {
			const payload: any = { email };
			if (password) payload.password = password;

			const updatedUser = await fetchApi('/users/profile', {
				method: 'PUT',
				body: JSON.stringify(payload)
			});
			
			userState.user = { ...userState.user, ...updatedUser };
			message = { type: 'success', text: 'Security settings updated successfully!' };
			password = '';
			confirmPassword = '';
		} catch (err: any) {
			message = { type: 'error', text: err.message || 'Failed to update security settings.' };
		} finally {
			isSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>My Profile - Aurahub</title>
</svelte:head>

<main class="mx-auto max-w-6xl space-y-8">
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Settings</h1>
			<p class="text-muted-foreground mt-1">
				Manage your public profile and account security.
			</p>
		</div>
	</div>

	<!-- Custom Tabs (No Sidebar) -->
	<div class="flex border-b">
		<button
			class="border-b-2 px-4 py-3 text-sm font-medium transition-colors {activeTab === 'profile'
				? 'border-primary text-primary'
				: 'text-muted-foreground border-transparent hover:border-border hover:text-primary'}"
			onclick={() => (activeTab = 'profile')}
		>
			<div class="flex items-center gap-2">
				<User class="h-4 w-4" />
				Public Profile
			</div>
		</button>
		<button
			class="border-b-2 px-4 py-3 text-sm font-medium transition-colors {activeTab === 'security'
				? 'border-primary text-primary'
				: 'text-muted-foreground border-transparent hover:border-border hover:text-primary'}"
			onclick={() => (activeTab = 'security')}
		>
			<div class="flex items-center gap-2">
				<Shield class="h-4 w-4" />
				Account Security
			</div>
		</button>
	</div>

	<!-- Main Content Area -->
	<div class="min-h-[400px]">
			{#if message.text}
				<div class={`mb-6 flex items-center gap-2 rounded-xl p-4 ${message.type === 'success' ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' : 'bg-rose-500/10 text-rose-600 dark:text-rose-400'}`}>
					{#if message.type === 'success'}
						<CheckCircle2 size={20} />
					{:else}
						<XCircle size={20} />
					{/if}
					<span class="font-medium">{message.text}</span>
				</div>
			{/if}

			<!-- PUBLIC PROFILE TAB -->
			{#if activeTab === 'profile'}
				<div class="fade-in space-y-8">
					<!-- Banner & Avatar Section -->
					<div class="bg-card border-border relative overflow-hidden rounded-2xl border shadow-sm">
						<!-- Banner -->
						<div class="bg-muted group relative h-32 w-full sm:h-48">
							{#if banner}
								<img src={banner} alt="Banner" class="h-full w-full object-cover" />
							{:else}
								<div class="flex h-full w-full items-center justify-center opacity-50">
									<ImageIcon size={40} class="text-muted-foreground" />
								</div>
							{/if}
							<label class="bg-foreground/40 absolute inset-0 flex cursor-pointer items-center justify-center opacity-0 transition-opacity duration-300 group-hover:opacity-100">
								<span class="text-primary-foreground bg-foreground/50 flex items-center gap-2 rounded-full px-4 py-2 font-medium backdrop-blur-sm">
									<ImageIcon size={18} /> Change Banner
								</span>
								<input type="file" accept="image/*" class="hidden" onchange={(e) => handleImageUpload(e, 'banner')} />
							</label>
						</div>

						<!-- Avatar -->
						<div class="relative z-10 flex items-end justify-between px-6 pb-6">
							<div class="group relative inline-block -mt-12 sm:-mt-16">
								<img
									src={avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${username}`}
									alt={username || 'Profile picture'}
									class="bg-card ring-background h-24 w-24 rounded-full object-cover ring-4 sm:h-32 sm:w-32"
								/>
								<label class="bg-foreground/50 absolute inset-0 flex cursor-pointer items-center justify-center rounded-full opacity-0 transition-opacity duration-300 group-hover:opacity-100">
									<span class="text-primary-foreground text-center text-xs font-medium">Change<br />Avatar</span>
									<input type="file" accept="image/*" class="hidden" onchange={(e) => handleImageUpload(e, 'avatar')} />
								</label>
							</div>
						</div>
					</div>

					<!-- Form Section -->
					<div class="bg-card border-border rounded-2xl border p-6 shadow-sm sm:p-8">
						<h2 class="text-foreground font-display mb-6 text-xl font-bold tracking-tight">Profile Details</h2>
						<form onsubmit={saveProfile} class="space-y-5">
							<div>
								<label class="text-muted-foreground mb-1 block text-sm font-medium">Username</label>
								<input
									type="text"
									bind:value={username}
									disabled
									title="Username cannot be changed"
									class="border-border bg-muted/50 text-muted-foreground w-full rounded-xl border px-4 py-3 opacity-70 cursor-not-allowed"
								/>
							</div>

							<div>
								<label class="text-muted-foreground mb-1 block text-sm font-medium">Bio</label>
								<textarea
									bind:value={bio}
									rows="4"
									placeholder="Tell viewers about your channel..."
									class="border-border bg-muted/50 text-foreground focus:ring-ring focus:bg-card w-full resize-none rounded-xl border px-4 py-3 transition-all focus:ring-2"
								></textarea>
							</div>

							<div class="flex justify-end pt-4">
								<button
									type="submit"
									disabled={isSubmitting}
									class="text-primary-foreground flex items-center gap-2 rounded-xl bg-primary px-6 py-2.5 font-medium shadow-sm transition-all hover:bg-primary/90 disabled:opacity-70"
								>
									{#if isSubmitting}<Loader2 class="animate-spin" size={18} />{/if}
									Save Profile
								</button>
							</div>
						</form>
					</div>
				</div>
			{/if}

			<!-- SECURITY TAB -->
			{#if activeTab === 'security'}
				<div class="fade-in">
					<div class="bg-card border-border rounded-2xl border p-6 shadow-sm sm:p-8">
						<h2 class="text-foreground font-display mb-2 text-xl font-bold tracking-tight">Account Security</h2>
						<p class="text-muted-foreground mb-6 text-sm">Manage your email and update your password.</p>

						<form onsubmit={saveSecurity} class="space-y-5">
							<div>
								<label class="text-muted-foreground mb-1 block text-sm font-medium">Email Address</label>
								<input
									type="email"
									bind:value={email}
									required
									class="border-border bg-muted/50 text-foreground focus:ring-ring focus:bg-card w-full rounded-xl border px-4 py-3 transition-all focus:ring-2"
								/>
							</div>

							<hr class="border-border my-6" />

							<h3 class="text-foreground mb-4 text-sm font-semibold uppercase tracking-wider">Change Password</h3>

							<div>
								<label class="text-muted-foreground mb-1 block text-sm font-medium">New Password (Optional)</label>
								<input
									type="password"
									bind:value={password}
									placeholder="Leave blank to keep current password"
									class="border-border bg-muted/50 text-foreground focus:ring-ring focus:bg-card w-full rounded-xl border px-4 py-3 transition-all focus:ring-2"
								/>
							</div>

							{#if password}
								<div class="fade-in duration-300">
									<label class="text-muted-foreground mb-1 block text-sm font-medium">Confirm New Password</label>
									<input
										type="password"
										bind:value={confirmPassword}
										required={!!password}
										class="border-border bg-muted/50 text-foreground focus:ring-ring focus:bg-card w-full rounded-xl border px-4 py-3 transition-all focus:ring-2"
									/>
								</div>
							{/if}

							<div class="flex justify-end pt-4">
								<button
									type="submit"
									disabled={isSubmitting}
									class="text-primary-foreground flex items-center gap-2 rounded-xl bg-primary px-6 py-2.5 font-medium shadow-sm transition-all hover:bg-primary/90 disabled:opacity-70"
								>
									{#if isSubmitting}<Loader2 class="animate-spin" size={18} />{/if}
									Save Security Settings
								</button>
							</div>
						</form>
					</div>
				</div>
			{/if}
		</div>
</main>

<style>
	.fade-in {
		animation: fadeIn 0.4s ease-in-out;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
</style>
