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
 * filename: sdi_entity_reset.c
 */


/**************************************************************************************
 * Implementation of entity reset and power status control APIs.
 ***************************************************************************************/

#include "sdi_entity.h"

/**
 * Reset the specified entity.
 * Reset of entity results in reset of resources and devices as per the reset type.
 * Upon reset, default configurations as specified for platform would be applied.
 * param[in] hdl - handle to the entity whose information has to be retrieved.
 * param[in] type – type of reset to perform.
 * return STD_ERR_OK on success and standard error on failure
 */
t_std_error sdi_entity_reset(sdi_entity_hdl_t hdl, sdi_reset_type_t type)
{
    return (STD_ERR_OK);
}

/**
 * Change/Control the power status for the specified entity.
 *
 * param[in] hdl - handle to the entity whose information has to be retrieved.
 * param[in] enable – power state to enable / disable
 * return STD_ERR_OK on success and standard error on failure
 */
t_std_error sdi_entity_power_status_control(sdi_entity_hdl_t hdl, bool enable)
{
    return (STD_ERR_OK);
}
