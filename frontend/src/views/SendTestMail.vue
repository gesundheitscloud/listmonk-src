<template>
    <form action="">
        <div class="modal-card" style="width: auto">
            <header class="modal-card-head">
                <p class="modal-card-title">{{ $t('templates.sendtest') }}</p>
            </header>
            <section class="modal-card-body">
                <b-field :label="$t('settings.smtp.toEmail')" label-position="on-border">
                  <b-input
                        type="email"
                        ref="inputTestEmail"
                        v-model="testEmail"
                        placeholder='email@site.com'
                        required>
                  </b-input>
                </b-field>
                <div v-if="errMsg">
                <b-field class="mt-4" type="is-danger">
                  <b-input v-model="errMsg" type="textarea"
                    custom-class="has-text-danger is-size-6" readonly />
                </b-field>
              </div>
            </section>
            <footer class="modal-card-foot">
                <b-button
                    :label="$t('globals.buttons.close')"
                    @click="close" />
                <b-button
                    :label="$t('globals.buttons.sendtest')"
                    :disabled="!isFormValid"
                    type="is-primary"
                    @click.prevent="sendTest" />
            </footer>
        </div>
    </form>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  props: {
    data: Object,
  },

  data() {
    return {
      testEmail: '',
      errMsg: '',
      isFormValid: false,
    };
  },

  watch: {
    testEmail(/* newVal, oldVal */) {
      this.isFormValid = this.$refs.inputTestEmail.checkHtml5Validity();
    },
  },

  methods: {
    close() {
      this.$emit('close');
    },

    sendTest() {
      this.errMsg = '';
      this.$api.sendTxSync({
        subscriber_email: this.testEmail,
        template_id: this.data.id,
      }).then(() => {
        this.$utils.toast(this.$t('campaigns.testSent'));
        this.close();
      }).catch((err) => {
        if (err.response?.data?.message) {
          this.errMsg = err.response.data.message;
        }
      });
    },
  },
});
</script>
