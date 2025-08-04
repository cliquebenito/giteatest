<template>
  <h4 class="ui top attached header">
    {{ $t('tenant.title.page') }}
    <button class="sc-button sc-button_primary"  @click="createTenantMode">{{ $t('tenant.createButton') }}</button>
  </h4>


  <div v-if="alert" class="ui message flash-message flash-success" :class="alert.type">
    <p>{{ alert.message }}</p>
  </div>


  <tenant-search v-model="filter" />

  <tenant-table
    :tenants="filteredTenants"
    :loading="loadingData"
    @editTenant="editTenantMode($event)"
  />


  <aside v-if="isPanelVisible" class="aside-panel">
    <div class="aside-panel-content">

      <div v-if="mode === 'create'" class="tenant-create">
        <tenant-create-panel
          :loading="loadingButton"
          @create="createTenant($event)"
          @cancelCreate="hideTenantPanel"
        />
      </div>

      <div v-else-if="mode === 'edit'" class="tenant-edit">
        <tenant-edit-panel
          :tenant="editTenant"
          :loading="loadingButton"
          @edit="updateTenant($event)"
          @cancelEdit="cancelEditTenant"
          @delete="deleteTenant"
        />

      </div>
    </div>
  </aside>



</template>

<script>
  import TenantSearch from './TenantSearch.vue';
  import TenantTable from './TenantTable.vue';
  import TenantCreatePanel from './TenantCreatePanel.vue';
  import TenantEditPanel from './TenantEditPanel.vue';
  import 'vue3-easy-data-table/dist/style.css';
  import { useI18n } from 'vue-i18n';

  const { pageData, csrfToken, lang } = window.config;
  const ALERT_TIMEOUT = 5000;

  export default  {
    components: {
      TenantSearch,
      TenantTable,
      TenantCreatePanel,
      TenantEditPanel,
    },
    data() {
      return ({
        isPanelVisible: false,
        mode: null,
        newTenant: {
          name: '',
          isActive: false
        },
        alert: null,
        timeoutId: null,
        editTenant: null,
        loadingData: false,
        loadingButton: false,
        tenants: [],
        filter: '',
        lang,
      })
    },

    created() {
      this.fetchTenants();
      document.addEventListener('keydown', this.handleCloseModal);
    },

    beforeDestroy() {
      document.removeEventListener('keydown', this.handleCloseModal);
      clearTimeout(this.timeoutId);
    },

    computed: {
      filteredTenants() {
        return this.tenants.filter(item => item.name.toLowerCase().includes(this.filter.toLowerCase()))
      }
    },

    setup() {
      const { t, locale } = useI18n();
      return { t, locale }
    },

    methods: {

      deleteTenant() {
        const formData = new FormData();
        formData.append('_csrf', csrfToken);

        this.fetchData(`/${this.editTenant.id}/delete`, formData, () => {
          this.editTenant = null;
          this.hideTenantPanel();
          this.fetchTenants();
        }, {
          success: 'tenant.message.success.delete',
          error: 'tenant.message.failure.delete'
        });
      },

      updateTenant(tenant) {
        const formData = new FormData();
        formData.append('_csrf', csrfToken);
        formData.append('name', tenant.name);
        formData.append('is_active', tenant.isActive);

        this.fetchData(`/${this.editTenant.id}/edit`, formData, () => {
          this.editTenant = null;
          this.hideTenantPanel();
          this.fetchTenants();
        }, {
          success: 'tenant.message.success.update',
          error: 'tenant.message.failure.update'
        });
      },

      createTenant(tenant) {
        const formData = new FormData();
        formData.append('_csrf', csrfToken);
        formData.append('name', tenant.name);
        formData.append('is_active', tenant.isActive);

        this.fetchData('/new', formData, () => {
          this.hideTenantPanel();
          tenant.callback();
          this.fetchTenants();
        }, {
          success: 'tenant.message.success.create',
          error: 'tenant.message.failure.create'
        });
      },

      createTenantMode() {
        document.body.classList.add('sidebar-panel-mode');
        this.isPanelVisible = true;
        this.mode = 'create';
      },

      editTenantMode(tenant) {
        document.body.classList.add('sidebar-panel-mode');
        this.isPanelVisible = true;
        this.mode = 'edit';

        this.editTenant = tenant;
      },

      hideTenantPanel() {
        document.body.classList.remove('sidebar-panel-mode');
        this.isPanelVisible = false;
        this.mode = null;
      },

      cancelEditTenant() {
        this.editTenant = null;
        this.hideTenantPanel();
      },


      handleCloseModal(event) {
        if (event.key === 'Escape') {
          if (this.mode === 'edit') {
            this.cancelEditTenant();
          } else if (this.mode === 'create') {
            this.hideTenantPanel();
          }
        }
      },

      async fetchTenants() {
        clearTimeout(this.timeoutId);
        this.loadingData = true;
        try {
          const response = await fetch(`${pageData.apiUrl}/list`, { method: 'GET' });
          if (!response.ok) {
            const error = await response.text()
            throw new Error(error)
          }
          const data = await response.json();

          const tenantsList = data.tenants.reduce((acc, item) => {
            const tenant = Object.values(item)[0][0];
            acc.push(tenant);
            return acc;
          }, []);

          this.tenants = tenantsList;
          this.loadingData = false;
        } catch (error) {
          this.alert = {
            type: 'negative',
            message: this.t('tenant.message.failure.fetch', { error })
          };
        } finally {
          this.timeoutId = setTimeout(() => {
            this.alert = null;
          }, ALERT_TIMEOUT)
        }

      },

      async fetchData(endpoint, formData, callback, message) {
        clearTimeout(this.timeoutId);
        this.loadingButton = true;
        try {
          let response = await fetch(`${pageData.apiUrl}${endpoint}`, { method: 'POST', body: formData });
          if (!response.ok) {
            const error = await response.text()
            throw new Error(error)
          }
          callback();
          this.alert = {
            type: 'positive',
            message: this.t(message.success, { name: formData.get('name') })
          };
        } catch (error) {
          this.hideTenantPanel();
           this.alert = {
             type: 'negative',
             message: this.t(message.error, { error })
           };
        } finally {
          this.loadingButton = false;
          this.timeoutId = setTimeout(() => {
            this.alert = null;
          }, ALERT_TIMEOUT)
        }
      },

    },

  }
</script>

<style scoped>
  .aside-panel {
    position: fixed;
    width: 443px;
    height: 100vh;
    background: var(--color-box-body);
    right: 0;
    top: 0;
    overflow: hidden;
    z-index: 1000;
  }

  .aside-panel::before {
    content: '';
    display: block;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    position: fixed;
    background: black;
    opacity: .4;
  }

  .aside-panel-content {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: var(--color-box-body);
    padding: 32px;
    box-sizing: border-box;
    overflow-y: auto;
    height: 100%;
  }

  .aside-panel-content.block::after {
    content: '';
    display: block;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: black;
    opacity: .4;
    z-index: 1001;
  }

  .ui.header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .tenant-create,
  .tenant-edit {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

</style>


