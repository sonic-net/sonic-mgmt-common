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
 * filename: sdi_thermal.c
 */


/**************************************************************************************
 * sdi_thermal.c
 * API implementation for thermal related functionalities
***************************************************************************************/

#include "sdi_thermal.h"

/*
 * API implementation to retrieve the temperature of the chip refered by resource.
 * [in] sensor_hdl - resource handle of the chip
 * [out] temp - temperature value is returned in this
 */
t_std_error sdi_temperature_get(sdi_resource_hdl_t sensor_hdl, int *temp)
{
    *temp = 0;                  /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/*
 * API implementation to retrieve the temperature thresholds of the chip refered by resource.
 * [in] sensor_hdl - resource handle of the chip
 * [in] threshold_type - type of the threshold(low/high)
 * [out] val - threshold value is returned in this
 */
t_std_error sdi_temperature_threshold_get(sdi_resource_hdl_t sensor_hdl,
        sdi_threshold_t threshold_type,  int *val)
{
    *val = 0;                  /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/*
 * API implementation to set the temperature thresholds of the chip refered by resource.
 * [in] sensor_hdl - resource handle of the chip
 * [in] threshold_type - type of the threshold(low/high)
 * [in] val - threshold value is returned in this
 */
t_std_error sdi_temperature_threshold_set(sdi_resource_hdl_t sensor_hdl,
        sdi_threshold_t threshold_type, int val)
{
    return (STD_ERR_OK);
}

/*
 * API implementation to retrieve the fault status of the chip refered by resource.
 * [in] sensor_hdl - resource handle of the chip
 * [out] alert_on - fault status is returned in this
 */
t_std_error sdi_temperature_status_get(sdi_resource_hdl_t sensor_hdl, bool *alert_on)
{
    *alert_on = false;                  /* Valid, but dummy value */

    return (STD_ERR_OK);
}
