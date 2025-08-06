<template>
  <h4 ref="title" class="ui top attached header panel-header">
    {{ $t('tenant.title.edit') }}

    <button class="sc-button sc-button_icon sc-button_transparent" @click.prevent="cancel">
        <svg-icon name="octicon-x"></svg-icon>
      </button>
  </h4>

  <form class="ui form">

    <div class="form__body">
      <div class="field">
        <input type="text" v-model="name">
      </div>

      <div class="field field_switch">
        <label for="isActive-edit">
          {{ $t('tenant.isActiveLabel') }}
        </label>
        <div class="ui toggle checkbox">
          <input type="checkbox" id="isActive-edit" name="isActive" v-model="isActive" :disabled="tenant.default">
          <label for="isActive-edit"></label>
        </div>
      </div>

      <p v-if="!isActive" class="hint">
        <svg-icon name="octicon-alert"></svg-icon>
        {{ $t('tenant.hint') }}
      </p>

      <div class="tenant-edit-projects">
        <h5 class="tenant-edit-projects__title">{{ $t('tenant.title.projects') }}</h5>
        <div v-if="projects" class="tenant-edit-projects__table-container">
          <easy-data-table
            :headers="headers"
            :items="projects"
            table-class-name="tenant-project-table"
            :body-item-class-name="getProjectNameClass"
            :hide-footer="true"
          >
            <template #item-CreatedUnix="{ CreatedUnix }">
              {{ new Date(CreatedUnix * 1000).toLocaleString(lang, { year: 'numeric', month: 'short', day: 'numeric' }) }}
            </template>
          </easy-data-table>
        </div>
        <p class="tenant-edit-projects__empty" v-else>
          <svg-icon name="sc-folder"></svg-icon>
          {{  $t('tenant.emptyProjects') }}
        </p>
      </div>

    </div>

    <div class="form__footer">
      <button @click.prevent="edit" :disabled="isEditButtonDisabled" class="sc-button sc-button_primary sc-button_fluid" :class="{'loading': loading }">
        {{ $t('buttons.save') }}
      </button>
      <button @click.prevent="cancel" class="sc-button sc-button_base sc-button_fluid">{{ $t('buttons.cancel') }}</button>
      <button v-if="!tenant.default" @click.prevent="showDialog" :disabled="isDeleteButtonDisabled" class="sc-button sc-button_danger sc-button_icon sc-button_icon-big" :data-tooltip-content="$t('tenant.tooltip.delete')">
        <svg-icon name="octicon-trash"></svg-icon>
      </button>
    </div>


    <div v-if="isVisible" class="dialog">
      <h4 class="ui top attached header dialog__header">
        {{ $t('tenant.title.delete') }}
        <button class="sc-button sc-button_icon dialog__close" @click.prevent="cancelDialog">
          <svg-icon name="octicon-x"></svg-icon>
        </button>
      </h4>
      <p class="dialog__body">
        {{ $t('tenant.deleteWarning', {name}) }}
      </p>
      <div class="dialog__footer">
        <button @click.prevent="remove" class="sc-button sc-button_fluid sc-button_primary" :class="{'loading': loading }">{{ $t('buttons.delete') }}</button>
        <button @click.prevent="cancelDialog" class="sc-button sc-button_fluid sc-button_base">{{ $t('buttons.cancel') }}</button>
      </div>
    </div>

  </form>
</template>

<script>
  import EasyDataTable from 'vue3-easy-data-table';
  import { ref } from 'vue';
  import { useI18n } from 'vue-i18n';

  export default {
    name: 'tenant-edit-panel',
    components: {
      EasyDataTable
    },

    props: {
      lang: {
        type: String,
        required: false
      },
      tenant: {
        type:  Object,
        required: true
      },

      loading: {
        type: Boolean,
        required: true
      }
    },

    emits: ['edit', 'cancelEdit', 'delete'],

    setup() {
      const { t, locale } = useI18n();
      return { t, locale }
    },


    data() {
      return ({
        isVisible: false,
        name: this.tenant.name,
        isActive: this.tenant.is_active,
        projects: this.tenant.projects,
        headers: [
          { text: this.t('tenant.tableProjectsHeaders.name'), value: 'Name', sortable: true  },
          { text: this.t('tenant.tableProjectsHeaders.create'), value: 'CreatedUnix' , sortable: true },
        ]
      })
    },

    computed: {
      isEditButtonDisabled() {
        return !this.name.length ||
          (this.name === this.tenant.name && this.tenant.is_active === this.isActive)
      },

      isDeleteButtonDisabled() {
        return this.isActive
      },
    },

    methods: {

      showDialog() {
        this.isVisible = true;
        this.$refs.title.parentElement.parentElement.classList.add('block');
      },

      cancelDialog() {
        this.isVisible = false;
        this.$refs.title.parentElement.parentElement.classList.remove('block');
      },

      cancel() {
        this.$emit('cancelEdit');
      },

      edit() {
        this.$emit('edit', { name: this.name, isActive: this.isActive });
      },

      remove() {
        this.$emit('delete');
      },


      getProjectNameClass(column) {
        return `tenant-project-table-${column.toLowerCase()}`
      },

    }
  }
</script>

<style scoped>

  .tenant-project-table {
    --easy-table-border: none;
    --easy-table-body-row-height: 50px;
    --easy-table-body-row-font-color: #1A5EE6;
  }

  .tenant-project-table-createdunix {
    color: #2E3038;
  }

  .theme-dark .tenant-project-table {
    --easy-table-header-font-color: var(--color-caret);
    --easy-table-header-background-color: var(--color-box-body);

    --easy-table-body-row-background-color: var(--color-box-body);
    --easy-table-body-even-row-font-color: #737B8C;
    --easy-table-body-row-font-color: #737B8C;

    --easy-table-body-row-hover-font-color: #737B8C;
    --easy-table-body-row-hover-background-color: #282e3f;
  }

  .override .ui.form .field.field_switch {
    display: flex;
    justify-content: space-between;
    flex-direction: row;
    align-items: center;
  }

  .panel-header {
    display: flex; 
    align-items: center;
    justify-content: space-between;
    padding: 0!important;
    margin-bottom: 24px!important;
  }

  .ui.toggle.checkbox {
    width: 40px;
  }

  .override .ui.toggle.checkbox label:before {
    background: var(--color-box-body);
  }


  .ui.form {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .ui .form__body {
    flex-grow: 1;
  }
  .ui .form__footer {
    display: flex;
    width: 100%;
    align-items: center;
    flex-direction: row;
    justify-content: space-between;
    margin-top: 24px;
    column-gap: 8px;
  }


  .tenant-edit .hint {
    display: flex;
    background: #FFD24C;
    color: #2E3038;
    padding: 10px;
    border-radius: 8px;
    font-size: 12px;
    column-gap: 10px;
    align-items: flex-start;
    margin-bottom: 24px;
  }

  .tenant-edit .hint svg {
    flex-shrink: 0;
    width: 16px;
    margin-top: 2px;
  }

  .tenant-edit .form__footer .button {
    width: 153px;
  }
  .tenant-edit .form__footer .button.icon {
    width: 40px;
    background: red;
    color: #fff;
    border-color: red;
  }
  .tenant-edit .form__footer .button.icon:hover {
    background-color: #d80909;
    border-color: #d80909;
  }

  .tenant-edit .form__footer .button.icon svg {
    color: inherit;
  }

  .tenant-edit .form__footer .button.icon:hover svg {
    color: #fff;
  }

  .tenant-edit-projects__title {
    font-size: 15px;
    margin-bottom: 12px;
  }

  .tenant-edit-projects__empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    color: #737B8C;
    font-size: 13px;
    row-gap: 12px;
    width: 220px;
    margin: 24px auto 0;
    text-align: center;
    
  }

  .tenant-edit-projects__empty svg {
    width: 69px;
    height: 133px;
  }

  .dialog {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 443px;
    padding: 24px;
    background: var(--color-box-body);
    border-radius: 24px;
    box-shadow: 0 1px 20px rgba(8, 27, 69, 0.2);
    z-index: 1002;
  }

  .dialog__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
  }

  .dialog__close {
    border: none;
    background: transparent;
    padding: 0;
    cursor: pointer;
  }

  .dialog__body {
    margin-bottom: 24px;
    font-size: 15px;
    line-height: 20px;
  }

  .dialog__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    column-gap: 8px;
  }

  .dialog__footer .button {
    width: 190px;
  }
</style>
