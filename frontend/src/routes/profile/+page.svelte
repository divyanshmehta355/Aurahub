<script lang="ts">
	import { onMount } from 'svelte';
	import { userState } from '#lib/user.svelte';
	import { fetchApi } from '#lib/api';
	import { Image as ImageIcon, Shield, User, CheckCircle2, XCircle, Loader2 } from 'lucide-svelte';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	import { Textarea } from '#lib/components/ui/textarea';
	import { Label } from '#lib/components/ui/label';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';

	const queryClient = useQueryClient();

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

	let message = $state({ type: '', text: '' });

	$effect(() => {
		if (userState.user) {
			username = userState.user.username;
			bio = userState.user.bio || '';
			avatar = userState.user.avatar || '';
			banner = userState.user.banner || '';
			email = userState.user.email || '';
		}
	});

	const imageUploadMutation = createMutation(() => ({
		mutationFn: async ({ file, type }: { file: File; type: 'avatar' | 'banner' }) => {
			const formData = new FormData();
			formData.append('image', file);
			formData.append('type', type);
			
			return await fetchApi('/user/profile/image', {
				method: 'POST',
				body: formData
			});
		},
		onSuccess: (data, { type }) => {
			if (type === 'avatar') {
				avatar = data.url;
				if (userState.user) userState.user.avatar = data.url;
			} else {
				banner = data.url;
				if (userState.user) userState.user.banner = data.url;
			}
			message = { type: 'success', text: `${type.charAt(0).toUpperCase() + type.slice(1)} updated successfully!` };
		},
		onError: (err: any) => {
			message = { type: 'error', text: err.message || 'Failed to upload image.' };
		}
	}), () => queryClient);

	async function handleImageUpload(e: Event, type: 'avatar' | 'banner') {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			const file = target.files[0];
			message = { type: '', text: '' };
			imageUploadMutation.mutate({ file, type });
		}
	}

	const saveProfileMutation = createMutation(() => ({
		mutationFn: async () => {
			return await fetchApi('/user/profile', {
				method: 'PUT',
				body: JSON.stringify({ bio })
			});
		},
		onSuccess: (res) => {
			if (userState.user) userState.user.bio = bio;
			message = { type: 'success', text: 'Profile updated successfully!' };
		},
		onError: (err: any) => {
			message = { type: 'error', text: err.message || 'Failed to update profile.' };
		}
	}), () => queryClient);

	const saveSecurityMutation = createMutation(() => ({
		mutationFn: async () => {
			if (password && password !== confirmPassword) {
				throw new Error('Passwords do not match');
			}
			const body: any = { email };
			if (password) body.password = password;
			
			return await fetchApi('/user/security', {
				method: 'PUT',
				body: JSON.stringify(body)
			});
		},
		onSuccess: (res) => {
			if (userState.user) userState.user.email = email;
			message = { type: 'success', text: 'Security settings updated successfully!' };
			password = '';
			confirmPassword = '';
		},
		onError: (err: any) => {
			message = { type: 'error', text: err.message || 'Failed to update security settings.' };
		}
	}), () => queryClient);

	function saveProfile(e: Event) {
		e.preventDefault();
		message = { type: '', text: '' };
		saveProfileMutation.mutate();
	}

	function saveSecurity(e: Event) {
		e.preventDefault();
		message = { type: '', text: '' };
		saveSecurityMutation.mutate();
	}
</script>

<svelte:head>
	<title>My Profile - Aurahub</title>
</svelte:head>

<main class="mx-auto max-w-4xl space-y-6 md:space-y-8 px-4 md:px-0 py-4 md:py-8">
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
		<div>
			<h1 class="text-2xl md:text-3xl font-bold tracking-tight">Settings</h1>
			<p class="text-muted-foreground mt-1 text-sm md:text-base">Manage your public profile and account security.</p>
		</div>
	</div>

	<!-- Custom Tabs (No Sidebar) -->
	<div class="flex border-b overflow-x-auto hide-scrollbar">
		<button
			class="border-b-2 px-4 py-3 text-sm font-medium transition-colors whitespace-nowrap {activeTab === 'profile'
				? 'border-primary text-primary'
				: 'text-muted-foreground border-transparent hover:border-border hover:text-primary'}"
			onclick={() => { activeTab = 'profile'; message = { type: '', text: '' }; }}
		>
			<div class="flex items-center gap-2">
				<User class="h-4 w-4" />
				Public Profile
			</div>
		</button>
		<button
			class="border-b-2 px-4 py-3 text-sm font-medium transition-colors whitespace-nowrap {activeTab === 'security'
				? 'border-primary text-primary'
				: 'text-muted-foreground border-transparent hover:border-border hover:text-primary'}"
			onclick={() => { activeTab = 'security'; message = { type: '', text: '' }; }}
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
			<div
				class={`mb-6 flex items-center gap-2 rounded-xl p-4 ${message.type === 'success' ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' : 'bg-destructive/10 text-destructive'}`}
			>
				{#if message.type === 'success'}
					<CheckCircle2 size={20} class="shrink-0" />
				{:else}
					<XCircle size={20} class="shrink-0" />
				{/if}
				<span class="font-medium text-sm md:text-base">{message.text}</span>
			</div>
		{/if}

		<!-- PUBLIC PROFILE TAB -->
		{#if activeTab === 'profile'}
			<div class="animate-in fade-in slide-in-from-bottom-2 duration-300 space-y-6 md:space-y-8">
				<!-- Banner & Avatar Section -->
				<div class="bg-card relative overflow-hidden rounded-2xl border shadow-sm">
					<!-- Banner -->
					<div class="bg-muted group relative h-32 w-full sm:h-48">
						{#if banner}
							<img src={banner} alt="Banner" class="h-full w-full object-cover" />
						{:else}
							<div class="flex h-full w-full items-center justify-center opacity-50">
								<ImageIcon size={40} class="text-muted-foreground" />
							</div>
						{/if}
						<label
							class="bg-foreground/40 absolute inset-0 flex cursor-pointer items-center justify-center opacity-0 transition-opacity duration-300 group-hover:opacity-100 {imageUploadMutation.isPending ? 'cursor-not-allowed opacity-100' : ''}"
						>
							{#if imageUploadMutation.isPending}
								<span class="text-primary-foreground bg-foreground/50 flex items-center gap-2 rounded-full px-4 py-2 font-medium backdrop-blur-sm">
									<Loader2 size={18} class="animate-spin" /> Uploading...
								</span>
							{:else}
								<span class="text-primary-foreground bg-foreground/50 flex items-center gap-2 rounded-full px-4 py-2 font-medium backdrop-blur-sm">
									<ImageIcon size={18} /> Change Banner
								</span>
								<input
									type="file"
									accept="image/*"
									class="hidden"
									onchange={(e) => handleImageUpload(e, 'banner')}
									disabled={imageUploadMutation.isPending}
								/>
							{/if}
						</label>
					</div>

					<!-- Avatar -->
					<div class="relative z-10 flex items-end justify-between px-4 sm:px-6 pb-4 sm:pb-6">
						<div class="group relative -mt-10 sm:-mt-16 inline-block">
							<img
								src={avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${username}`}
								alt={username || 'Profile picture'}
								class="bg-card h-20 w-20 sm:h-32 sm:w-32 rounded-full object-cover ring-4 ring-background"
							/>
							<label
								class="bg-foreground/50 absolute inset-0 flex cursor-pointer items-center justify-center rounded-full opacity-0 transition-opacity duration-300 group-hover:opacity-100 {imageUploadMutation.isPending ? 'cursor-not-allowed' : ''}"
							>
								{#if !imageUploadMutation.isPending}
									<span class="text-primary-foreground text-center text-[10px] sm:text-xs font-medium">Change<br />Avatar</span>
									<input
										type="file"
										accept="image/*"
										class="hidden"
										onchange={(e) => handleImageUpload(e, 'avatar')}
										disabled={imageUploadMutation.isPending}
									/>
								{/if}
							</label>
						</div>
					</div>
				</div>

				<!-- Form Section -->
				<div class="bg-card rounded-2xl border p-4 sm:p-6 md:p-8 shadow-sm">
					<h2 class="text-foreground mb-4 sm:mb-6 text-lg sm:text-xl font-bold tracking-tight">
						Profile Details
					</h2>
					<form onsubmit={saveProfile} class="space-y-4 sm:space-y-5">
						<div class="space-y-1.5">
							<Label>Username</Label>
							<Input
								type="text"
								bind:value={username}
								disabled
								title="Username cannot be changed"
								class="bg-muted/50 text-muted-foreground cursor-not-allowed opacity-70"
							/>
						</div>

						<div class="space-y-1.5">
							<Label>Bio</Label>
							<Textarea
								bind:value={bio}
								rows={4}
								placeholder="Tell viewers about your channel..."
								class="resize-none"
							/>
						</div>

						<div class="flex justify-end pt-2 sm:pt-4">
							<Button
								type="submit"
								disabled={saveProfileMutation.isPending}
								class="w-full sm:w-auto rounded-full px-6"
							>
								{#if saveProfileMutation.isPending}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								Save Profile
							</Button>
						</div>
					</form>
				</div>
			</div>
		{/if}

		<!-- SECURITY TAB -->
		{#if activeTab === 'security'}
			<div class="animate-in fade-in slide-in-from-bottom-2 duration-300">
				<div class="bg-card rounded-2xl border p-4 sm:p-6 md:p-8 shadow-sm">
					<h2 class="text-foreground mb-1 md:mb-2 text-lg sm:text-xl font-bold tracking-tight">
						Account Security
					</h2>
					<p class="text-muted-foreground mb-6 text-xs sm:text-sm">
						Manage your email and update your password.
					</p>

					<form onsubmit={saveSecurity} class="space-y-4 sm:space-y-5">
						<div class="space-y-1.5">
							<Label>Email Address</Label>
							<Input
								type="email"
								bind:value={email}
								required
							/>
						</div>

						<hr class="my-6 border-border" />

						<h3 class="text-foreground mb-3 sm:mb-4 text-xs sm:text-sm font-semibold tracking-wider uppercase">
							Change Password
						</h3>

						<div class="space-y-1.5">
							<Label>New Password (Optional)</Label>
							<Input
								type="password"
								bind:value={password}
								placeholder="Leave blank to keep current password"
							/>
						</div>

						{#if password}
							<div class="animate-in fade-in slide-in-from-top-2 space-y-1.5">
								<Label>Confirm New Password</Label>
								<Input
									type="password"
									bind:value={confirmPassword}
									required={!!password}
								/>
							</div>
						{/if}

						<div class="flex justify-end pt-2 sm:pt-4">
							<Button
								type="submit"
								disabled={saveSecurityMutation.isPending}
								class="w-full sm:w-auto rounded-full px-6"
							>
								{#if saveSecurityMutation.isPending}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								Save Security Settings
							</Button>
						</div>
					</form>
				</div>
			</div>
		{/if}
	</div>
</main>

<style>
	.hide-scrollbar::-webkit-scrollbar {
		display: none;
	}
	.hide-scrollbar {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
</style>
