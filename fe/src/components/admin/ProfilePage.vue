<script setup lang="ts">
import { ref } from 'vue';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { formatDateTime } from '@/lib/utils';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { User } from '@lucide/vue';

const { isReady, allowed, authStore } = usePageGuard();
const ui = useUiStore();
const t = useLocaleStore().t;
</script>

<template>
  <DashboardShell>
    <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
    </div>

    <div v-else class="flex flex-col gap-6 max-w-2xl">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.profile.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.profile.description') }}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2">
            <User class="h-5 w-5" />
            {{ t('admin.profile.title') }}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div class="grid gap-4">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="space-y-1">
                <div class="text-sm text-muted-foreground">{{ t('admin.profile.fullName') }}</div>
                <div class="font-medium">{{ authStore.currentUser?.full_name || '—' }}</div>
              </div>
              <div class="space-y-1">
                <div class="text-sm text-muted-foreground">{{ t('admin.profile.username') }}</div>
                <div class="font-medium">{{ authStore.currentUser?.username || '—' }}</div>
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="space-y-1">
                <div class="text-sm text-muted-foreground">{{ t('admin.profile.email') }}</div>
                <div class="font-medium">{{ authStore.currentUser?.email || '—' }}</div>
              </div>
              <div class="space-y-1">
                <div class="text-sm text-muted-foreground">{{ t('admin.profile.role') }}</div>
                <div class="font-medium">
                  <Badge variant="secondary">{{ authStore.currentUser?.role_name || '—' }}</Badge>
                </div>
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="space-y-1">
                <div class="text-sm text-muted-foreground">{{ t('admin.profile.status') }}</div>
                <div class="font-medium">
                  <Badge :variant="authStore.currentUser?.is_active ? 'default' : 'destructive'">
                    {{ authStore.currentUser?.is_active ? t('admin.profile.active') : t('admin.profile.inactive') }}
                  </Badge>
                </div>
              </div>
              <div class="space-y-1">
                <div class="text-sm text-muted-foreground">{{ t('admin.profile.memberSince') }}</div>
                <div class="font-medium">{{ authStore.currentUser?.created_at ? formatDateTime(authStore.currentUser.created_at) : '—' }}</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </DashboardShell>
</template>
