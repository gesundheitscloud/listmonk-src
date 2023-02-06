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
                        :value="testEmail"
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
                    @click="$emit('close')" />
                <b-button
                    :label="$t('globals.buttons.sendtest')"
                    type="is-primary"
                    @click.prevent="sendTest(data.id, testEmail)"/>
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
    };
  },

  methods: {
    close() {
      this.$emit('close');
    },

    sendTest(id, email) {
      this.errMsg = '';
      console.log(id, email, this.testEmail);
      this.$api.sendTxSync({
        subscriber_email: email,
        template_id: id,
      }).then(() => {
        this.$utils.toast(this.$t('campaigns.testSent'));
      }).catch((err) => {
        if (err.response?.data?.message) {
          this.errMsg = err.response.data.message;
        }
      });
    },
  },
});
</script>
