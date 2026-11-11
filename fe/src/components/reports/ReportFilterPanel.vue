<script setup lang="ts">
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import { Card, CardContent } from '@/components/ui/card';
import { useLocaleStore } from '@/stores/locale';

const localeStore = useLocaleStore();
const t = localeStore.t;

const fromDate = defineModel<string>('fromDate', { default: '' });
const toDate = defineModel<string>('toDate', { default: '' });

const emit = defineEmits<{
  apply: [];
}>();

function apply() {
  emit('apply');
}
</script>

<template>
  <Card>
    <CardContent class="grid grid-cols-1 gap-3 pt-4 sm:grid-cols-2 md:grid-cols-[auto_auto_1fr] md:items-end">
      <div class="space-y-1">
        <Label>{{ t('admin.reports.fromDate') }}</Label>
        <Input v-model="fromDate" type="date" class="w-full sm:w-40" />
      </div>
      <div class="space-y-1">
        <Label>{{ t('admin.reports.toDate') }}</Label>
        <Input v-model="toDate" type="date" class="w-full sm:w-40" />
      </div>
      <div class="flex sm:justify-start md:justify-end">
        <Button variant="outline" class="w-full sm:w-auto" @click="apply">{{ t('admin.reports.apply') }}</Button>
      </div>
    </CardContent>
  </Card>
</template>
