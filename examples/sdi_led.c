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
 * filename: sdi_led.c
 */


/**************************************************************************************
 *  Implementation of LED resource API.
 ***************************************************************************************/

#include "sdi_led.h"

/**
 * Turn-on the specified LED
 *
 * resource_hdl[in] - Handle of the Resource
 *
 * return t_std_error
 */
t_std_error sdi_led_on (sdi_resource_hdl_t resource_hdl)
{
    return (STD_ERR_OK);
}

/**
 * Turn-off the specified LED
 *
 * resource_hdl[in] - Handle of the resource
 *
 * return t_std_error
 */
t_std_error sdi_led_off (sdi_resource_hdl_t resource_hdl)
{
    return (STD_ERR_OK);
}

/**
 * Turn-on the digital display LED
 *
 * resource_hdl[in] - Handle of the LED
 *
 * return t_std_error
 */
t_std_error sdi_digital_display_led_on (sdi_resource_hdl_t resource_hdl)
{
    return (STD_ERR_OK);
}

/**
 * Turn-off the digital display LED
 *
 * resource_hdl[in] - Handle of the LED
 *
 * return t_std_error
 */
t_std_error sdi_digital_display_led_off (sdi_resource_hdl_t resource_hdl)
{
    return (STD_ERR_OK);
}

/**
 * Sets the specified value in the digital_display_led.
 *
 * hdl[in]           : Handle of the resource
 * display_string[in]: Value to be displayed
 *
 * return t_std_error
 */
t_std_error sdi_digital_display_led_set (sdi_resource_hdl_t hdl, const char *display_string)
{
    return (STD_ERR_OK);
}
