<template>
  <div class="table-wrapper">
    <easy-data-table
      ref="dataTable"
      :headers="headers"
      :items="paginatedTenants"
      :loading="loading"
      theme-color="#1A5EE6"
      :body-row-class-name="getTenantRowClasses"
      table-class-name="tenant-table"
      alternating
      :hide-footer="true"
    >
      <template #item-name="tenant">
        <div class="tenant-table__name">
          <svg-icon v-if="tenant.default" name="octicon-lock" class="tenant-row__icon_default"/>
          <svg-icon v-else-if="tenant.is_active" name="octicon-check-circle" class="tenant-row__icon_active"/>
          <svg-icon v-else-if="!tenant.is_active" name="octicon-dots-circle" class="tenant-row__icon_inactive"/>
          {{ tenant.name }}
        </div>
      </template>

      <template #item-created_at="{ created_at }">
        {{ new Date(created_at * 1000).toLocaleString(lang, { year: 'numeric', month: 'short', day: 'numeric' }) }}
      </template>

      <template #item-projects="{ projects }">
        {{ projects ? projects.length : 0 }}
      </template>

      <template #item-edit="tenant">
        <button class="sc-button sc-button_icon sc-button_transparent" :data-tooltip-content="$t('tenant.tooltip.edit')" @click="edit(tenant)">
          <svg-icon name="octicon-pencil"/>
        </button>
      </template>

      <template #empty-message>
        <p class="empty-message">
          {{ $t('tenant.emptyTable') }}
        </p>
      </template>
    </easy-data-table>
    <div class="pagination">
      <button
        v-for="paginationNumber in totalPages"
        :key="paginationNumber"
        class="sc-button sc-button_icon"
        :class="{'sc-button_primary': paginationNumber === currentPage,  'sc-button_transparent': paginationNumber !== currentPage}"
        @click="updatePage(paginationNumber)"
      >
        {{ paginationNumber }}
      </button>
    </div>
  </div>

</template>

<script>
import EasyDataTable from 'vue3-easy-data-table';
import { ref, computed } from 'vue';
import {useI18n} from 'vue-i18n';

const { lang: language } = window.config;
const dataTable = ref(null);

export default {
  name: 'TenantTable',

  components: {
    EasyDataTable,
  },

  props: {
    lang: {
      type: String,
      required: false,
    },
    tenants: {
      type: Array,
      required: true
    },
    loading: {
      type: Boolean,
      required: true
    }
  },

  emits: ['editTenant'],

  setup() {
    const {t, locale} = useI18n();
    return {t, locale};
  },

  data() {
    return {
      headers: [
        {text: this.t('tenant.tableTenantsHeaders.name'), value: 'name', sortable: true},
        {text: this.t('tenant.tableTenantsHeaders.projects'), value: 'projects', sortable: true},
        {text: this.t('tenant.tableTenantsHeaders.created'), value: 'created_at', sortable: true},
        {text: this.t('tenant.tableTenantsHeaders.edit'), value: 'edit'}
      ],
      itemsPerPage: 12,
      currentPage: 1,
    };
  },

  computed: {
    paginatedTenants() {
      const start = (this.currentPage - 1) * this.itemsPerPage;
      return this.tenants.slice(start, start + this.itemsPerPage);
    },
    totalPages() {
      return Math.ceil(this.tenants.length / this.itemsPerPage);
    }
  },

  methods: {
    edit(tenant) {
      this.$emit('editTenant', tenant);
    },

    updatePage(page) {
      if (page >= 1 && page <= this.totalPages) {
        this.currentPage = page;
      }
    },

    getTenantRowClasses(tenant) {
      const classNames = ['tenant-row'];
      if (tenant.default) {
        classNames.push('tenant-row_default');
      } else if (tenant.is_active) {
        classNames.push('tenant-row_active');
      } else if (!tenant.is_active) {
        classNames.push('tenant-row_inactive');
      }
      return classNames.join(' ');
    },
  },
};
</script>


<style scoped>
.tenant-table {
  --easy-table-border: none;
  --easy-table-body-row-height: 50px;
  --easy-table-body-row-font-color: #737B8C;
  --easy-table-header-font-size: 13px;
  --easy-table-body-row-font-size: 13px;
  --easy-table-body-row-hover-font-color: inherit;
  --easy-table-loading-mask-background-color: var(--color-primary);
}

.theme-dark .tenant-table {
  --easy-table-header-font-color: var(--color-caret);
  --easy-table-header-background-color: var(--color-box-body);
  --easy-table-body-row-background-color: var(--color-box-body);
  --easy-table-body-row-hover-background-color: #282e3f;
}

.tenant-table__name {
  display: flex;
  align-items: center;
  column-gap: 4px;
}

.tenant-row__icon_active {
  color: #66BB6A;
}
.tenant-row__icon_default {
  color: #66BB6A;
}
.tenant-row__icon_inactive {
  color: #78909C;
}

.table-wrapper {
  height: 70vh;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.pagination {
  display: flex;
  justify-content: center;
  margin: 20px 0;
}

.pagination__button {
  padding: 8px 16px;
  margin: 0 4px;
  border-radius: 8px;
  background-color: white;
  cursor: pointer;
  font-size: 14px;
  color: #737B8C;
  transition: background-color 0.3s ease;
}

.pagination__button:hover {
  background-color: #f0f0f0;
}

.pagination__button--active {
  background-color: #f0f0f0;
  color: black;
}

.pagination__button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>

