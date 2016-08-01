/*
 * Copyright (c) 2016 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * THIS CODE IS PROVIDED ON AN  *AS IS* BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING WITHOUT
 *  LIMITATION ANY IMPLIED WARRANTIES OR CONDITIONS OF TITLE, FITNESS
 * FOR A PARTICULAR PURPOSE, MERCHANTABLITY OR NON-INFRINGEMENT.
 *
 * See the Apache Version 2.0 License for specific language governing
 * permissions and limitations under the License.
 */

/*
 * filename: sdi_media.c
 */


/*******************************************************************************
 * Implementation of Media resource API.
 ******************************************************************************/

#include "sdi_media.h"
#include <string.h>
/**
 * Get the present status of the specific media
 * resource_hdl[in] - Handle of the resource
 * pres[out]        - "true" if module is present else "false"
 * return t_std_error
 */
t_std_error sdi_media_presence_get (sdi_resource_hdl_t resource_hdl, bool *pres)
{
    *pres = false;              /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Gets the required module monitors(temperature and voltage) alarm status
 * resource_hdl[in] - Hnadle of the resource
 * flags[in]        - flags for status that are of interest
 * status[out]      - returns the set of status flags
 * return t_std_error
 */
t_std_error sdi_media_module_monitor_status_get (sdi_resource_hdl_t resource_hdl,
                                                 uint_t flags, uint_t *status)
{
    *status = 0;                /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Get the required channel monitoring(rx_power and tx_bias) alarm status of media.
 * resource_hdl[in] - Handle of the resource
 * channel[in]      - channel number that is of interest, it should be '0' if
 *                    only one channel is present
 * flags[in]        - flags for channel status
 * status[out]      - returns the set of status flags which are asserted.
 * return           - standard t_std_error
 */
t_std_error sdi_media_channel_monitor_status_get (sdi_resource_hdl_t resource_hdl, uint_t channel,
                                                  uint_t flags, uint_t *status)
{
    *status = 0;                /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Get the required channel status of the specific media.
 * resource_hdl[in] - Handle of the resource
 * channel[in]      - channel number that is of interest, it should be '0' if
 *                    only one channel is present
 * flags[in]        - flags for channel status
 * status[out]      - returns the set of status flags which are asserted.
 * return           - standard t_std_error
 */
t_std_error sdi_media_channel_status_get (sdi_resource_hdl_t resource_hdl, uint_t channel,
                                          uint_t flags, uint_t *status)
{
    *status = 0;                /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Disable/Enable the transmitter of the specific media.
 * resource_hdl[in] - handle of the media resource
 * channel[in]      - channel number that is of interest and should be 0 only
 *                    one channel is present
 * enable[in]       - "false" to disable and "true" to enable
 * @return          - standard t_std_error
 */
t_std_error sdi_media_tx_control (sdi_resource_hdl_t resource_hdl, uint_t channel,
                                  bool enable)
{
    return (STD_ERR_OK);
}

/**
 * For getting transmitter status(enabled/disabled) on the specified channel
 * resource_hdl[in] - handle of the media resource
 * channel[in]      - channel number
 * status[out]      -  transmitter status-> "true" if enabled, else "false"
 * return           - standard t_std_error
 */
t_std_error sdi_media_tx_control_status_get (sdi_resource_hdl_t resource_hdl,
                                             uint_t channel, bool *status)
{
    *status = 0;                /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Get the maximum speed that can be supported by a specific media resource
 * resource_hdl[in] - handle of the media resource
 * speed[out]       - maximum speed that can be supported by media device
 * return           - standard t_std_error
 */
t_std_error sdi_media_speed_get (sdi_resource_hdl_t resource_hdl,
                                 sdi_media_speed_t *speed)
{
    *speed = SDI_MEDIA_SPEED_10M;  /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Checks whether the specified media is qualified by DELL or not
 * resource_hdl[in] - handle of the media resource
 * status[out]      - "true" if media is qualified by DELL else "false"
 * return           - standard t_std_error
 */
t_std_error sdi_media_is_dell_qualified (sdi_resource_hdl_t resource_hdl,
                                         bool *status)
{
    *status = false;            /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Reads the requested parameter value from eeprom
 * resource_hdl[in] - handle of the media resource
 * param[in]        - parametr type that is of interest(e.g wavelength, maximum
 *                    case temperature etc)
 * value[out]       - parameter value which is read from eeprom
 * return           - standard t_std_error
 */
t_std_error sdi_media_parameter_get (sdi_resource_hdl_t resource_hdl,
                                     sdi_media_param_type_t param, uint_t *value)
{
    *value = 0;                 /* Dummy value */

    return (STD_ERR_OK);
}

/**
 * Read the requested vendor information of a specific media resource
 * resource_hdl[in]     - handle of the media resource
 * vendor_info_type[in] - vendor information that is of interest.
 * vendor_info[out]     - vendor information which is read from eeprom
 * buf_size[in]         - size of the input buffer(vendor_info)
 * return               - standard t_std_error
 */
t_std_error sdi_media_vendor_info_get (sdi_resource_hdl_t resource_hdl,
                                       sdi_media_vendor_info_type_t vendor_info_type,
                                       char *vendor_info, size_t buf_size)
{
    memset(vendor_info, 0, buf_size); /* Dummy values */

    return (STD_ERR_OK);
}

/**
 * Read the transceiver compliance code of a specific media resource
 * resource_hdl[in]     - handle of the media resource
 * transceiver_info[out]- transceiver compliance code which is read from eeprom
 * return               - standard t_std_error
 */
t_std_error sdi_media_transceiver_code_get (sdi_resource_hdl_t resource_hdl,
                                            sdi_media_transceiver_descr_t *transceiver_info)
{
    memset(transceiver_info, 0, sizeof(*transceiver_info)); /* Dummy values */

    return (STD_ERR_OK);
}

/**
 * Read the dell product information
 * resource_hdl[in] - Handle of the resource
 * info[out] - dell product information
 * return - standard t_std_error
 */
t_std_error sdi_media_dell_product_info_get (sdi_resource_hdl_t resource_hdl,
                                             sdi_media_dell_product_info_t *info)
{
    memset(info, 0, sizeof(*info)); /* Dummy values */

    return (STD_ERR_OK);
}

/**
 * Get the alarm and warning threshold values for a given optics
 * resource_hdl[in] - Handle of the resource
 * threshold_type[in] - type of threshold
 * value[out] - threshold value
 * return - standard t_std_error
 */
t_std_error sdi_media_threshold_get (sdi_resource_hdl_t resource_hdl,
                                     sdi_media_threshold_type_t threshold_type,
                                     float *value)
{
    *value = 0.0;               /* Dummy value */

    return (STD_ERR_OK);
}

/**
 * Read the threshold values for module monitors like temperature and voltage
 * resource_hdl[in] - Handle of the resource
 * threshold_type[in] - type of threshold
 * value[out] - threshold value
 * return - standard t_std_error
 * TODO: depricated API. Need to remove once upper layers adopted new api
 * sdi_media_threshold_get
 */
t_std_error sdi_media_module_monitor_threshold_get (sdi_resource_hdl_t resource_hdl,
                                                    uint_t threshold_type, uint_t *value)
{
    *value = 0;                 /* Dummy value */

    return (STD_ERR_OK);
}

/**
 * Read the threshold values for channel monitors like rx-ower and tx-bias
 * resource_hdl[in] - Handle of the resource
 * threshold_type[in] - type of threshold
 * value[out] - threshold value
 * return - standard t_std_error
 * TODO: depricated API. Need to remove once upper layers adopted new api
 * sdi_media_threshold_get
 */
t_std_error sdi_media_channel_monitor_threshold_get (sdi_resource_hdl_t resource_hdl,
                                                     uint_t threshold_type, uint_t *value)
{
    *value = 0;                 /* Dummy value */

    return (STD_ERR_OK);
}

/**
 * Enable/Disable the module control parameters like low power mode and reset
 * control
 * resource_hdl[in] - handle of the resource
 * ctrl_type[in]    - module control type(LP mode/reset)
 * enable[in]       - "true" to enable and "false" to disable
 * return           - standard t_std_error
 */
t_std_error sdi_media_module_control (sdi_resource_hdl_t resource_hdl,
                                      sdi_media_module_ctrl_type_t ctrl_type, bool enable)
{
    return (STD_ERR_OK);
}

/**
 * Enable/Disable Auto neg on SFP PHY
 * @param[in] resource_hdl - handle to media
 * @param[in] channel - channel number that is of interest. channel numbers
 * should start with 0(e.g. For qsfp valid channel number range is 0 to 3) and
 * channel number should be '0' if only one channel is present
 * @param[in] type - media type which is present in front panel port
 * @param[in] enable - true-enable Autonge,false-disable Autoneg.
 * @return - standard @ref t_std_error
 *
 */
t_std_error sdi_media_phy_autoneg_set (sdi_resource_hdl_t resource_hdl, uint_t channel,
                                       sdi_media_type_t type, bool enable)
{
    return (STD_ERR_OK);
}

/**
 * set mode on  SFP PHY
 * @param[in] resource_hdl - handle to media
 * @param[in] channel - channel number that is of interest. channel numbers
 * should start with 0(e.g. For qsfp valid channel number range is 0 to 3) and
 * channel number should be '0' if only one channel is present
 * @param[in] type - media type which is present in front panel port
 * @param[in] mode - SGMII/MII/GMII, Should be of type Refer @ref sdi_media_mode_t.
 * @return - standard @ref t_std_error
 *
 */
t_std_error sdi_media_phy_mode_set (sdi_resource_hdl_t resource_hdl,
                                    uint_t channel, sdi_media_type_t type,
                                    sdi_media_mode_t mode)
{
    return (STD_ERR_OK);
}

/**
 * set speed on  SFP PHY
 * @param[in] resource_hdl - handle to media
 * @param[in] channel - channel number that is of interest. channel numbers
 * should start with 0(e.g. For qsfp valid channel number range is 0 to 3) and
 * channel number should be '0' if only one channel is present
 * @param[in] type - media type which is present in front panel port
 * @param[in] speed - phy supported speed's. Should be of type @ref sdi_media_speed_t .
   @count[in] count - count for number of phy supported speed's 10/100/1000.
 * @return - standard @ref t_std_error
 *
 */
t_std_error sdi_media_phy_speed_set (sdi_resource_hdl_t resource_hdl,
                                     uint_t channel, sdi_media_type_t type,
                                     sdi_media_speed_t *speed, uint_t count)
{
    return (STD_ERR_OK);
}

/**
 * Get the status of module control parameters like low power mode and reset
 * status
 * resource_hdl[in] - handle of the resource
 * ctrl_type[in]    - module control type(LP mode/reset)
 * status[out]      - "true" if enabled else "false"
 * return           - standard t_std_error
 */
t_std_error sdi_media_module_control_status_get (sdi_resource_hdl_t resource_hdl,
                                                 sdi_media_module_ctrl_type_t ctrl_type,
                                                 bool *status)
{
    *status = false;            /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Retrieve module monitors assoicated with the specified media
 * resource_hdl[in] - handle of the media resource
 * monitor[in]      - monitor which needs to be retrieved(TEMPERATURE/VOLTAGE)
 * value[out]       - Value of the monitor
 * return           - standard t_std_error
 */
t_std_error sdi_media_module_monitor_get (sdi_resource_hdl_t resource_hdl,
                                          sdi_media_module_monitor_t monitor, float *value)
{
    *value = 0.0;               /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Retrieve channel monitors assoicated with the specified media.
 * resource_hdl[in] - handle of the media resource
 * channel[in]      - channel whose monitor has to be retreived and should be 0,
 *                    if only one channel is present.
 * monitor[in]      - monitor which needs to be retrieved.
 * value[out]       - Value of the monitor
 * return           - standard t_std_error
 */
t_std_error sdi_media_channel_monitor_get (sdi_resource_hdl_t resource_hdl,
                                           uint_t channel, sdi_media_channel_monitor_t monitor, float *value)
{
    *value = 0.0;               /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Read data from media
 * resource_hdl[in] - handle of the media resource
 * offset[in]       - offset from which data to be read
 * data[out]        - buffer for data to be read
 * data_len[in]     - length of the data to be read
 * return           - standard t_std_error
 */
t_std_error sdi_media_read (sdi_resource_hdl_t resource_hdl, uint_t offset,
                            uint8_t *data, size_t data_len)
{
    memset(data, 0, data_len);  /* Dummy values */

    return (STD_ERR_OK);
}

/**
 * Write data to media
 * resource_hdl[in] - handle of the media resource
 * offset[in]       - offset to which data to be write
 * data[in]         - input buffer which contains data to be written
 * data_len[in]     - length of the data to be write
 * return standard t_std_error
 */
t_std_error sdi_media_write (sdi_resource_hdl_t resource_hdl, uint_t offset,
                             uint8_t *data, size_t data_len)
{
    return (STD_ERR_OK);
}

/**
 * Get the optional feature support status on a given optics
 * resource_hdl[in] - handle of the media resource
 * feature_support[out] - feature support flags. Flag will be set to "true" if
 * feature is supported else "false"
 * return - standard t_std_error
 */
t_std_error sdi_media_feature_support_status_get (sdi_resource_hdl_t resource_hdl,
                                                  sdi_media_supported_feature_t *feature_support)
{
    memset(feature_support, 0, sizeof(*feature_support)); /* Dummy values */

    return (STD_ERR_OK);
}

/**
 * Set the port LED based on the speed settings of the port.
 * resource_hdl[in] - Handle of the resource
 * channel[in] - Channel number. Should be 0, if only one channel is present
 * speed[in] - LED mode setting is derived from speed
 * return - standard t_std_error
 */
t_std_error sdi_media_led_set (sdi_resource_hdl_t resource_hdl, uint_t channel,
                               sdi_media_speed_t speed)
{
    return (STD_ERR_OK);
}
