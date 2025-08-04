<template>
  <h4 class="ui top attached header panel-header">
    {{ $t('tenant.title.create') }}

    <button class="sc-button sc-button_icon sc-button_transparent" @click.prevent="cancel">
      <svg-icon name="octicon-x"/>
    </button>
  </h4>

  <form class="ui form">
    <div class="form__body">
      <div class="form-row">
        <div class="field">
          <input type="text" tabindex="1" autofocus :placeholder="$t('tenant.placeholder.create')" v-model="name">
        </div>
      </div>
    </div>

    <div class="form__footer">
      <button @click.prevent="create" :disabled="isCreateButtonDisabled" class="sc-button sc-button_primary sc-button_fluid" :class="{loading: loading }">
        {{ $t('buttons.save') }}
      </button>
      <button @click.prevent="cancel" class="sc-button sc-button_base sc-button_fluid">
        {{ $t('buttons.cancel') }}
      </button>
    </div>
  </form>
</template>

<script>
import EasyDataTable from 'vue3-easy-data-table';
import {SvgIcon} from '../../svg';

export default {
  name: 'TenantCreatePanel',

  components: {
    EasyDataTable,
    SvgIcon
  },

  props: {
    loading: {
      type: Boolean,
      required: true
    }
  },

  emits: ['create', 'cancelCreate'],

  data() {
    return ({
      name: '',
      isActive: true
    });
  },

  computed: {
    isCreateButtonDisabled() {
      return !this.name.length;
    },
  },

  methods: {
    create() {
      this.$emit('create', {
        name: this.name,
        isActive: this.isActive,
        callback: () => {
          this.name = '';
          this.isActive = false;
        }
      });
    },

    cancel() {
      this.name = '';
      this.isActive = false;
      this.$emit('cancelCreate');
    },
  }
};
</script>

<style scoped>

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0!important;
  margin-bottom: 24px!important;
}

  .override .ui.form .field.field_switch {
    display: flex;
    justify-content: space-between;
    flex-direction: row;
    align-items: center;
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
  .tenant-create .form__footer .button {
    width: 177px;
  }
</style>
